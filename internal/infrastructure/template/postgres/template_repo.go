package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"

	domaintemplate "github.com/vladgrskkh/onerep-api/internal/domain/template"
	"github.com/vladgrskkh/onerep-api/internal/infrastructure/postgresutil"
)

const (
	argTemplateID = "template_id"
	argSortOrder  = "sort_order"
)

type TemplateRepo struct {
	db     trmpgx.Tr
	getter *trmpgx.CtxGetter
}

func NewTemplateRepo(pool *pgxpool.Pool) *TemplateRepo {
	return &TemplateRepo{db: pool, getter: trmpgx.DefaultCtxGetter}
}

func (r *TemplateRepo) List(
	ctx context.Context,
	filter domaintemplate.TemplateFilter,
) ([]*domaintemplate.Template, error) {
	query, args := buildTemplateListQuery(filter)
	rows, err := r.getter.DefaultTrOrDB(ctx, r.db).Query(ctx, query, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*domaintemplate.Template
	for rows.Next() {
		t := &domaintemplate.Template{}
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

func buildTemplateListQuery(filter domaintemplate.TemplateFilter) (string, pgx.NamedArgs) {
	var query strings.Builder
	query.WriteString(
		`SELECT id, name, description, is_public, created_by_user_id, created_at, updated_at, version FROM gym.templates WHERE deleted_at IS NULL`,
	)
	args := pgx.NamedArgs{}

	if filter.UserID != uuid.Nil {
		query.WriteString(` AND created_by_user_id = @user_id`)
		args["user_id"] = filter.UserID
	}
	if filter.IsPublic != nil {
		query.WriteString(` AND is_public = @is_public`)
		args["is_public"] = *filter.IsPublic
	}
	if !filter.Since.IsZero() {
		query.WriteString(` AND updated_at > @since`)
		args["since"] = filter.Since
	}
	query.WriteString(` ORDER BY updated_at DESC`)

	return query.String(), args
}

func (r *TemplateRepo) FindByID(ctx context.Context, id uuid.UUID) (*domaintemplate.Template, error) {
	conn := r.getter.DefaultTrOrDB(ctx, r.db)
	rows, err := conn.Query(ctx, `
		SELECT t.id, t.name, t.description, t.is_public, t.created_by_user_id,
		       t.created_at, t.updated_at, t.deleted_at, t.version,
		       te.exercise_id, te.sort_order, te.planned_sets,
		       m.id, m.media_type, m.sort_order, m.s3_key
		FROM gym.templates t
		LEFT JOIN gym.template_exercises te ON te.template_id = t.id
		LEFT JOIN gym.template_media m ON m.template_id = t.id
		WHERE t.id = @id AND t.deleted_at IS NULL
		ORDER BY te.sort_order, m.sort_order
	`, pgx.NamedArgs{"id": id})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		t             *domaintemplate.Template
		exercisesSeen = make(map[uuid.UUID]struct{})
		mediaSeen     = make(map[uuid.UUID]struct{})
	)
	for rows.Next() {
		if t == nil {
			t = &domaintemplate.Template{}
		}
		var (
			deletedAt     *time.Time
			teExerciseID  *uuid.UUID
			teSortOrder   *int
			tePlannedSets *int
			mID           *uuid.UUID
			mMediaType    *string
			mSortOrder    *int
			mS3Key        *string
		)
		if scanErr := rows.Scan(
			&t.ID, &t.Name, &t.Description, &t.IsPublic, &t.CreatedByUserID,
			&t.CreatedAt, &t.UpdatedAt, &deletedAt, &t.Version,
			&teExerciseID, &teSortOrder, &tePlannedSets,
			&mID, &mMediaType, &mSortOrder, &mS3Key,
		); scanErr != nil {
			return nil, scanErr
		}
		t.DeletedAt = postgresutil.ValueOrNilTime(deletedAt)
		if teExerciseID != nil {
			if _, seen := exercisesSeen[*teExerciseID]; !seen {
				t.Exercises = append(t.Exercises, domaintemplate.TemplateExercise{
					TemplateID:  t.ID,
					ExerciseID:  *teExerciseID,
					SortOrder:   *teSortOrder,
					PlannedSets: *tePlannedSets,
				})
				exercisesSeen[*teExerciseID] = struct{}{}
			}
		}
		if mID != nil {
			if _, seen := mediaSeen[*mID]; !seen {
				t.Media = append(t.Media, domaintemplate.TemplateMedia{
					ID:         *mID,
					TemplateID: t.ID,
					MediaType:  domaintemplate.MediaType(*mMediaType),
					SortOrder:  *mSortOrder,
					S3Key:      *mS3Key,
				})
				mediaSeen[*mID] = struct{}{}
			}
		}
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, rowsErr
	}
	if t == nil {
		return nil, domaintemplate.ErrTemplateNotFound
	}
	return t, nil
}

func (r *TemplateRepo) Create(ctx context.Context, t domaintemplate.Template) (*domaintemplate.Template, error) {
	conn := r.getter.DefaultTrOrDB(ctx, r.db)
	if _, err := conn.Exec(ctx, `
		INSERT INTO gym.templates (id, name, description, is_public, created_by_user_id, created_at, updated_at, version)
		VALUES (@id, @name, @description, @is_public, @created_by_user_id, @created_at, @updated_at, @version)
	`, pgx.NamedArgs{
		"id":                 t.ID,
		"name":               t.Name,
		"description":        t.Description,
		"is_public":          t.IsPublic,
		"created_by_user_id": t.CreatedByUserID,
		"created_at":         t.CreatedAt,
		"updated_at":         t.UpdatedAt,
		"version":            t.Version,
	}); err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TemplateRepo) InsertExercise(ctx context.Context, te domaintemplate.TemplateExercise) error {
	conn := r.getter.DefaultTrOrDB(ctx, r.db)
	_, err := conn.Exec(ctx, `
		INSERT INTO gym.template_exercises (template_id, exercise_id, sort_order, planned_sets)
		VALUES (@template_id, @exercise_id, @sort_order, @planned_sets)
	`, pgx.NamedArgs{
		argTemplateID:  te.TemplateID,
		"exercise_id":  te.ExerciseID,
		argSortOrder:   te.SortOrder,
		"planned_sets": te.PlannedSets,
	})
	return err
}

func (r *TemplateRepo) BatchInsertExercises(
	ctx context.Context,
	exercises []domaintemplate.TemplateExercise,
) error {
	if len(exercises) == 0 {
		return nil
	}
	conn := r.getter.DefaultTrOrDB(ctx, r.db)
	batch := &pgx.Batch{}
	for _, te := range exercises {
		batch.Queue(`
			INSERT INTO gym.template_exercises (template_id, exercise_id, sort_order, planned_sets)
			VALUES (@template_id, @exercise_id, @sort_order, @planned_sets)
		`, pgx.NamedArgs{
			argTemplateID:  te.TemplateID,
			"exercise_id":  te.ExerciseID,
			argSortOrder:   te.SortOrder,
			"planned_sets": te.PlannedSets,
		})
	}
	return conn.SendBatch(ctx, batch).Close()
}

func (r *TemplateRepo) ReplaceExercises(
	ctx context.Context,
	templateID uuid.UUID,
	exercises []domaintemplate.TemplateExercise,
) error {
	conn := r.getter.DefaultTrOrDB(ctx, r.db)
	if _, err := conn.Exec(ctx, `
		DELETE FROM gym.template_exercises WHERE template_id = @template_id
	`, pgx.NamedArgs{argTemplateID: templateID}); err != nil {
		return err
	}
	return r.BatchInsertExercises(ctx, exercises)
}

func (r *TemplateRepo) InsertMedia(ctx context.Context, m domaintemplate.TemplateMedia) error {
	conn := r.getter.DefaultTrOrDB(ctx, r.db)
	_, err := conn.Exec(ctx, `
		INSERT INTO gym.template_media (id, template_id, media_type, sort_order, s3_key)
		VALUES (@id, @template_id, @media_type, @sort_order, @s3_key)
	`, pgx.NamedArgs{
		"id":          m.ID,
		argTemplateID: m.TemplateID,
		"media_type":  m.MediaType,
		argSortOrder:  m.SortOrder,
		"s3_key":      m.S3Key,
	})
	return err
}

func (r *TemplateRepo) BatchInsertMedia(ctx context.Context, media []domaintemplate.TemplateMedia) error {
	if len(media) == 0 {
		return nil
	}
	conn := r.getter.DefaultTrOrDB(ctx, r.db)
	batch := &pgx.Batch{}
	for _, m := range media {
		batch.Queue(`
			INSERT INTO gym.template_media (id, template_id, media_type, sort_order, s3_key)
			VALUES (@id, @template_id, @media_type, @sort_order, @s3_key)
		`, pgx.NamedArgs{
			"id":          m.ID,
			argTemplateID: m.TemplateID,
			"media_type":  m.MediaType,
			argSortOrder:  m.SortOrder,
			"s3_key":      m.S3Key,
		})
	}
	return conn.SendBatch(ctx, batch).Close()
}

func (r *TemplateRepo) ReplaceMedia(
	ctx context.Context,
	templateID uuid.UUID,
	media []domaintemplate.TemplateMedia,
) error {
	conn := r.getter.DefaultTrOrDB(ctx, r.db)
	if _, err := conn.Exec(ctx, `
		DELETE FROM gym.template_media WHERE template_id = @template_id
	`, pgx.NamedArgs{argTemplateID: templateID}); err != nil {
		return err
	}
	return r.BatchInsertMedia(ctx, media)
}

func (r *TemplateRepo) Update(ctx context.Context, t domaintemplate.Template) (*domaintemplate.Template, error) {
	conn := r.getter.DefaultTrOrDB(ctx, r.db)

	t.Version++
	tag, err := conn.Exec(ctx, `
		UPDATE gym.templates SET
			name = @name, description = @description, is_public = @is_public,
			updated_at = @updated_at, version = @version
		WHERE id = @id AND version = @expected_version
	`, pgx.NamedArgs{
		"name":             t.Name,
		"description":      t.Description,
		"is_public":        t.IsPublic,
		"updated_at":       t.UpdatedAt,
		"version":          t.Version,
		"id":               t.ID,
		"expected_version": t.Version - 1,
	})
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, domaintemplate.ErrTemplateNotFound
	}
	return &t, nil
}

func (r *TemplateRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	conn := r.getter.DefaultTrOrDB(ctx, r.db)
	_, err := conn.Exec(ctx, `
		UPDATE gym.templates SET deleted_at = now(), updated_at = now(), version = version + 1
		WHERE id = @id
	`, pgx.NamedArgs{"id": id})
	return err
}
