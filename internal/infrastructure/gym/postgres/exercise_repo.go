package postgres

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domainexercise "github.com/vladgrskkh/onerep-api/internal/domain/exercise"
	serviceexercise "github.com/vladgrskkh/onerep-api/internal/service/exercise"
)

type ExerciseRepo struct {
	pool *pgxpool.Pool
}

func NewExerciseRepo(pool *pgxpool.Pool) *ExerciseRepo {
	return &ExerciseRepo{pool: pool}
}

func (r *ExerciseRepo) List(
	ctx context.Context,
	filter serviceexercise.ExerciseFilter,
) ([]domainexercise.Exercise, error) {
	query, args := buildListQuery(filter)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exercises []domainexercise.Exercise
	for rows.Next() {
		var ex domainexercise.Exercise
		if scanErr := rows.Scan(
			&ex.ID,
			&ex.Name,
			&ex.Description,
			&ex.Notes,
			&ex.IsBuiltIn,
			&ex.CreatedByUserID,
			&ex.CreatedAt,
			&ex.UpdatedAt,
			&ex.Version,
		); scanErr != nil {
			return nil, scanErr
		}
		exercises = append(exercises, ex)
	}
	return exercises, rows.Err()
}

func buildListQuery(filter serviceexercise.ExerciseFilter) (string, []any) {
	var query strings.Builder
	query.WriteString(
		`SELECT id, name, description, notes, is_built_in, created_by_user_id, created_at, updated_at, version FROM gym.exercises WHERE deleted_at IS NULL`,
	)
	var args []any

	if filter.Search != "" {
		args = append(args, filter.Search)
		query.WriteString(" AND name ILIKE '%'||$" + strconv.Itoa(len(args)) + "||'%'")
	}
	if filter.MuscleGroup != "" {
		args = append(args, filter.MuscleGroup)
		query.WriteString(
			" AND EXISTS (SELECT 1 FROM gym.exercise_muscle_groups emg JOIN gym.muscle_groups mg ON mg.id = emg.muscle_group_id WHERE emg.exercise_id = exercises.id AND mg.name = $" + strconv.Itoa(
				len(args),
			) + ")",
		)
	}
	if filter.IsBuiltIn != nil {
		args = append(args, *filter.IsBuiltIn)
		query.WriteString(" AND is_built_in = $" + strconv.Itoa(len(args)))
	}
	if filter.Since != nil {
		args = append(args, *filter.Since)
		query.WriteString(" AND updated_at > $" + strconv.Itoa(len(args)))
	}
	query.WriteString(" ORDER BY name")

	return query.String(), args
}

func (r *ExerciseRepo) FindByID(ctx context.Context, id uuid.UUID) (domainexercise.Exercise, error) {
	var ex domainexercise.Exercise
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, description, notes, is_built_in, created_by_user_id, created_at, updated_at, deleted_at, version
		FROM gym.exercises WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&ex.ID, &ex.Name, &ex.Description, &ex.Notes, &ex.IsBuiltIn, &ex.CreatedByUserID, &ex.CreatedAt, &ex.UpdatedAt, &ex.DeletedAt, &ex.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return domainexercise.Exercise{}, domainexercise.ErrExerciseNotFound
	}
	if err != nil {
		return domainexercise.Exercise{}, err
	}

	ex.Media, err = r.findMedia(ctx, ex.ID)
	if err != nil {
		return domainexercise.Exercise{}, err
	}
	ex.MuscleGroups, err = r.findMuscleGroups(ctx, ex.ID)
	if err != nil {
		return domainexercise.Exercise{}, err
	}
	return ex, nil
}

func (r *ExerciseRepo) findMedia(ctx context.Context, exerciseID uuid.UUID) ([]domainexercise.ExerciseMedia, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, exercise_id, media_type, sort_order, s3_key
		FROM gym.exercise_media WHERE exercise_id = $1 ORDER BY sort_order
	`, exerciseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var media []domainexercise.ExerciseMedia
	for rows.Next() {
		var m domainexercise.ExerciseMedia
		if scanErr := rows.Scan(&m.ID, &m.ExerciseID, &m.MediaType, &m.SortOrder, &m.S3Key); scanErr != nil {
			return nil, scanErr
		}
		media = append(media, m)
	}
	return media, rows.Err()
}

func (r *ExerciseRepo) findMuscleGroups(
	ctx context.Context,
	exerciseID uuid.UUID,
) ([]domainexercise.ExerciseMuscleGroup, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT emg.exercise_id, emg.muscle_group_id, emg.is_primary
		FROM gym.exercise_muscle_groups emg
		JOIN gym.muscle_groups mg ON mg.id = emg.muscle_group_id
		WHERE emg.exercise_id = $1
		ORDER BY emg.is_primary DESC
	`, exerciseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var muscleGroups []domainexercise.ExerciseMuscleGroup
	for rows.Next() {
		var mg domainexercise.ExerciseMuscleGroup
		if scanErr := rows.Scan(&mg.ExerciseID, &mg.MuscleGroupID, &mg.IsPrimary); scanErr != nil {
			return nil, scanErr
		}
		muscleGroups = append(muscleGroups, mg)
	}
	return muscleGroups, rows.Err()
}

func (r *ExerciseRepo) Create(ctx context.Context, ex domainexercise.Exercise) (domainexercise.Exercise, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domainexercise.Exercise{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err = tx.Exec(ctx, `
		INSERT INTO gym.exercises (id, name, description, notes, is_built_in, created_by_user_id, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, ex.ID, ex.Name, ex.Description, ex.Notes, ex.IsBuiltIn, ex.CreatedByUserID, ex.CreatedAt, ex.UpdatedAt, ex.Version); err != nil {
		return domainexercise.Exercise{}, err
	}

	for _, m := range ex.Media {
		if _, err = tx.Exec(ctx, `
			INSERT INTO gym.exercise_media (id, exercise_id, media_type, sort_order, s3_key)
			VALUES ($1, $2, $3, $4, $5)
		`, m.ID, m.ExerciseID, m.MediaType, m.SortOrder, m.S3Key); err != nil {
			return domainexercise.Exercise{}, err
		}
	}

	for _, mg := range ex.MuscleGroups {
		if _, err = tx.Exec(ctx, `
			INSERT INTO gym.exercise_muscle_groups (exercise_id, muscle_group_id, is_primary)
			VALUES ($1, $2, $3)
		`, mg.ExerciseID, mg.MuscleGroupID, mg.IsPrimary); err != nil {
			return domainexercise.Exercise{}, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return domainexercise.Exercise{}, err
	}
	return ex, nil
}

func (r *ExerciseRepo) Update(ctx context.Context, ex domainexercise.Exercise) (domainexercise.Exercise, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domainexercise.Exercise{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ex.Version++
	tag, err := tx.Exec(ctx, `
		UPDATE gym.exercises SET
			name = $1, description = $2, notes = $3,
			updated_at = $4, version = $5
		WHERE id = $6 AND version = $7
	`, ex.Name, ex.Description, ex.Notes, ex.UpdatedAt, ex.Version, ex.ID, ex.Version-1)
	if err != nil {
		return domainexercise.Exercise{}, err
	}
	if tag.RowsAffected() == 0 {
		return domainexercise.Exercise{}, domainexercise.ErrExerciseNotFound
	}

	if _, err = tx.Exec(ctx, `DELETE FROM gym.exercise_media WHERE exercise_id = $1`, ex.ID); err != nil {
		return domainexercise.Exercise{}, err
	}
	for _, m := range ex.Media {
		if _, err = tx.Exec(ctx, `
			INSERT INTO gym.exercise_media (id, exercise_id, media_type, sort_order, s3_key)
			VALUES ($1, $2, $3, $4, $5)
		`, m.ID, m.ExerciseID, m.MediaType, m.SortOrder, m.S3Key); err != nil {
			return domainexercise.Exercise{}, err
		}
	}

	if _, err = tx.Exec(ctx, `DELETE FROM gym.exercise_muscle_groups WHERE exercise_id = $1`, ex.ID); err != nil {
		return domainexercise.Exercise{}, err
	}
	for _, mg := range ex.MuscleGroups {
		if _, err = tx.Exec(ctx, `
			INSERT INTO gym.exercise_muscle_groups (exercise_id, muscle_group_id, is_primary)
			VALUES ($1, $2, $3)
		`, mg.ExerciseID, mg.MuscleGroupID, mg.IsPrimary); err != nil {
			return domainexercise.Exercise{}, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return domainexercise.Exercise{}, err
	}
	return ex, nil
}

func (r *ExerciseRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE gym.exercises SET deleted_at = now(), updated_at = now(), version = version + 1
		WHERE id = $1
	`, id)
	return err
}
