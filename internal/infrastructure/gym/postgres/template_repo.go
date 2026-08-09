package postgres

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domaintemplate "github.com/vladgrskkh/onerep-api/internal/domain/template"
	servicetemplate "github.com/vladgrskkh/onerep-api/internal/service/template"
)

type TemplateRepo struct {
	pool *pgxpool.Pool
}

func NewTemplateRepo(pool *pgxpool.Pool) *TemplateRepo {
	return &TemplateRepo{pool: pool}
}

func (r *TemplateRepo) List(
	ctx context.Context,
	filter servicetemplate.TemplateFilter,
) ([]domaintemplate.Template, error) {
	query, args := buildTemplateListQuery(filter)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []domaintemplate.Template
	for rows.Next() {
		var t domaintemplate.Template
		if scanErr := rows.Scan(
			&t.ID,
			&t.Name,
			&t.Description,
			&t.IsPublic,
			&t.CreatedByUserID,
			&t.CreatedAt,
			&t.UpdatedAt,
			&t.Version,
		); scanErr != nil {
			return nil, scanErr
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

func buildTemplateListQuery(filter servicetemplate.TemplateFilter) (string, []any) {
	var query strings.Builder
	query.WriteString(
		`SELECT id, name, description, is_public, created_by_user_id, created_at, updated_at, version FROM gym.templates WHERE deleted_at IS NULL`,
	)
	var args []any

	if filter.UserID != nil {
		args = append(args, *filter.UserID)
		query.WriteString(" AND created_by_user_id = $" + strconv.Itoa(len(args)))
	}
	if filter.IsPublic != nil {
		args = append(args, *filter.IsPublic)
		query.WriteString(" AND is_public = $" + strconv.Itoa(len(args)))
	}
	if filter.Since != nil {
		args = append(args, *filter.Since)
		query.WriteString(" AND updated_at > $" + strconv.Itoa(len(args)))
	}
	query.WriteString(" ORDER BY updated_at DESC")

	return query.String(), args
}

func (r *TemplateRepo) FindByID(ctx context.Context, id uuid.UUID) (domaintemplate.Template, error) {
	var t domaintemplate.Template
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, description, is_public, created_by_user_id, created_at, updated_at, deleted_at, version
		FROM gym.templates WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&t.ID, &t.Name, &t.Description, &t.IsPublic, &t.CreatedByUserID, &t.CreatedAt, &t.UpdatedAt, &t.DeletedAt, &t.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return domaintemplate.Template{}, domaintemplate.ErrTemplateNotFound
	}
	if err != nil {
		return domaintemplate.Template{}, err
	}

	t.Exercises, err = r.findExercises(ctx, t.ID)
	if err != nil {
		return domaintemplate.Template{}, err
	}
	t.Media, err = r.findMedia(ctx, t.ID)
	if err != nil {
		return domaintemplate.Template{}, err
	}
	return t, nil
}

func (r *TemplateRepo) findExercises(
	ctx context.Context,
	templateID uuid.UUID,
) ([]domaintemplate.TemplateExercise, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT template_id, exercise_id, sort_order, planned_sets
		FROM gym.template_exercises WHERE template_id = $1 ORDER BY sort_order
	`, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exercises []domaintemplate.TemplateExercise
	for rows.Next() {
		var te domaintemplate.TemplateExercise
		if scanErr := rows.Scan(&te.TemplateID, &te.ExerciseID, &te.SortOrder, &te.PlannedSets); scanErr != nil {
			return nil, scanErr
		}
		exercises = append(exercises, te)
	}
	return exercises, rows.Err()
}

func (r *TemplateRepo) findMedia(ctx context.Context, templateID uuid.UUID) ([]domaintemplate.TemplateMedia, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, template_id, media_type, sort_order, s3_key
		FROM gym.template_media WHERE template_id = $1 ORDER BY sort_order
	`, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var media []domaintemplate.TemplateMedia
	for rows.Next() {
		var tm domaintemplate.TemplateMedia
		if scanErr := rows.Scan(&tm.ID, &tm.TemplateID, &tm.MediaType, &tm.SortOrder, &tm.S3Key); scanErr != nil {
			return nil, scanErr
		}
		media = append(media, tm)
	}
	return media, rows.Err()
}

func (r *TemplateRepo) Create(ctx context.Context, t domaintemplate.Template) (domaintemplate.Template, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domaintemplate.Template{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err = tx.Exec(ctx, `
		INSERT INTO gym.templates (id, name, description, is_public, created_by_user_id, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, t.ID, t.Name, t.Description, t.IsPublic, t.CreatedByUserID, t.CreatedAt, t.UpdatedAt, t.Version); err != nil {
		return domaintemplate.Template{}, err
	}

	for _, te := range t.Exercises {
		if _, err = tx.Exec(ctx, `
			INSERT INTO gym.template_exercises (template_id, exercise_id, sort_order, planned_sets)
			VALUES ($1, $2, $3, $4)
		`, te.TemplateID, te.ExerciseID, te.SortOrder, te.PlannedSets); err != nil {
			return domaintemplate.Template{}, err
		}
	}

	for _, m := range t.Media {
		if _, err = tx.Exec(ctx, `
			INSERT INTO gym.template_media (id, template_id, media_type, sort_order, s3_key)
			VALUES ($1, $2, $3, $4, $5)
		`, m.ID, m.TemplateID, m.MediaType, m.SortOrder, m.S3Key); err != nil {
			return domaintemplate.Template{}, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return domaintemplate.Template{}, err
	}
	return t, nil
}

func (r *TemplateRepo) Update(ctx context.Context, t domaintemplate.Template) (domaintemplate.Template, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domaintemplate.Template{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	t.Version++
	tag, err := tx.Exec(ctx, `
		UPDATE gym.templates SET
			name = $1, description = $2, is_public = $3,
			updated_at = $4, version = $5
		WHERE id = $6 AND version = $7
	`, t.Name, t.Description, t.IsPublic, t.UpdatedAt, t.Version, t.ID, t.Version-1)
	if err != nil {
		return domaintemplate.Template{}, err
	}
	if tag.RowsAffected() == 0 {
		return domaintemplate.Template{}, domaintemplate.ErrTemplateNotFound
	}

	if _, err = tx.Exec(ctx, `DELETE FROM gym.template_exercises WHERE template_id = $1`, t.ID); err != nil {
		return domaintemplate.Template{}, err
	}
	for _, te := range t.Exercises {
		if _, err = tx.Exec(ctx, `
			INSERT INTO gym.template_exercises (template_id, exercise_id, sort_order, planned_sets)
			VALUES ($1, $2, $3, $4)
		`, te.TemplateID, te.ExerciseID, te.SortOrder, te.PlannedSets); err != nil {
			return domaintemplate.Template{}, err
		}
	}

	if _, err = tx.Exec(ctx, `DELETE FROM gym.template_media WHERE template_id = $1`, t.ID); err != nil {
		return domaintemplate.Template{}, err
	}
	for _, m := range t.Media {
		if _, err = tx.Exec(ctx, `
			INSERT INTO gym.template_media (id, template_id, media_type, sort_order, s3_key)
			VALUES ($1, $2, $3, $4, $5)
		`, m.ID, m.TemplateID, m.MediaType, m.SortOrder, m.S3Key); err != nil {
			return domaintemplate.Template{}, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return domaintemplate.Template{}, err
	}
	return t, nil
}

func (r *TemplateRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE gym.templates SET deleted_at = now(), updated_at = now(), version = version + 1
		WHERE id = $1
	`, id)
	return err
}
