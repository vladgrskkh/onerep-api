package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"

	domainexercise "github.com/vladgrskkh/onerep-api/internal/domain/exercise"
)

const argExerciseID = "exercise_id"

type ExerciseRepo struct {
	db     trmpgx.Tr
	getter *trmpgx.CtxGetter
}

func NewExerciseRepo(pool *pgxpool.Pool) *ExerciseRepo {
	return &ExerciseRepo{db: pool, getter: trmpgx.DefaultCtxGetter}
}

func (r *ExerciseRepo) List(
	ctx context.Context,
	filter domainexercise.ExerciseFilter,
) ([]domainexercise.Exercise, error) {
	query, args := buildExerciseListQuery(filter)
	rows, err := r.getter.DefaultTrOrDB(ctx, r.db).Query(ctx, query, args)
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

func buildExerciseListQuery(filter domainexercise.ExerciseFilter) (string, pgx.NamedArgs) {
	var query strings.Builder
	query.WriteString(
		`SELECT id, name, description, notes, is_built_in, created_by_user_id, created_at, updated_at, version FROM gym.exercises WHERE deleted_at IS NULL`,
	)
	args := pgx.NamedArgs{}

	if filter.Search != "" {
		query.WriteString(` AND name ILIKE '%'||@search||'%'`)
		args["search"] = filter.Search
	}
	if filter.MuscleGroup != "" {
		query.WriteString(`
			AND EXISTS (SELECT 1 FROM gym.exercise_muscle_groups emg
				JOIN gym.muscle_groups mg ON mg.id = emg.muscle_group_id
				WHERE emg.exercise_id = exercises.id AND mg.name = @muscle_group)`)
		args["muscle_group"] = filter.MuscleGroup
	}
	if filter.IsBuiltIn != nil {
		query.WriteString(` AND is_built_in = @is_built_in`)
		args["is_built_in"] = *filter.IsBuiltIn
	}
	if filter.Since != nil {
		query.WriteString(` AND updated_at > @since`)
		args["since"] = *filter.Since
	}
	query.WriteString(` ORDER BY name`)

	return query.String(), args
}

func (r *ExerciseRepo) FindByID(ctx context.Context, id uuid.UUID) (domainexercise.Exercise, error) {
	conn := r.getter.DefaultTrOrDB(ctx, r.db)
	rows, err := conn.Query(ctx, `
		SELECT e.id, e.name, e.description, e.notes, e.is_built_in, e.created_by_user_id,
		       e.created_at, e.updated_at, e.deleted_at, e.version,
		       m.id, m.media_type, m.sort_order, m.s3_key,
		       emg.muscle_group_id, emg.is_primary
		FROM gym.exercises e
		LEFT JOIN gym.exercise_media m ON m.exercise_id = e.id
		LEFT JOIN gym.exercise_muscle_groups emg ON emg.exercise_id = e.id
		WHERE e.id = @id AND e.deleted_at IS NULL
		ORDER BY m.sort_order, emg.is_primary DESC
	`, pgx.NamedArgs{"id": id})
	if err != nil {
		return domainexercise.Exercise{}, err
	}
	defer rows.Close()

	var (
		ex         domainexercise.Exercise
		mediaSeen  = make(map[uuid.UUID]struct{})
		groupsSeen = make(map[int]struct{})
	)
	for rows.Next() {
		var (
			mID         *uuid.UUID
			mMediaType  *string
			mSortOrder  *int
			mS3Key      *string
			mgID        *int
			mgIsPrimary *bool
		)
		if scanErr := rows.Scan(
			&ex.ID, &ex.Name, &ex.Description, &ex.Notes, &ex.IsBuiltIn, &ex.CreatedByUserID,
			&ex.CreatedAt, &ex.UpdatedAt, &ex.DeletedAt, &ex.Version,
			&mID, &mMediaType, &mSortOrder, &mS3Key,
			&mgID, &mgIsPrimary,
		); scanErr != nil {
			return domainexercise.Exercise{}, scanErr
		}
		if mID != nil {
			if _, seen := mediaSeen[*mID]; !seen {
				ex.Media = append(ex.Media, domainexercise.ExerciseMedia{
					ID:         *mID,
					ExerciseID: ex.ID,
					MediaType:  domainexercise.MediaType(*mMediaType),
					SortOrder:  *mSortOrder,
					S3Key:      *mS3Key,
				})
				mediaSeen[*mID] = struct{}{}
			}
		}
		if mgID != nil {
			if _, seen := groupsSeen[*mgID]; !seen {
				ex.MuscleGroups = append(ex.MuscleGroups, domainexercise.ExerciseMuscleGroup{
					ExerciseID:    ex.ID,
					MuscleGroupID: *mgID,
					IsPrimary:     *mgIsPrimary,
				})
				groupsSeen[*mgID] = struct{}{}
			}
		}
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return domainexercise.Exercise{}, rowsErr
	}
	if ex.ID == uuid.Nil {
		return domainexercise.Exercise{}, domainexercise.ErrExerciseNotFound
	}
	return ex, nil
}

func (r *ExerciseRepo) Create(ctx context.Context, ex domainexercise.Exercise) (domainexercise.Exercise, error) {
	conn := r.getter.DefaultTrOrDB(ctx, r.db)
	if _, err := conn.Exec(ctx, `
		INSERT INTO gym.exercises (id, name, description, notes, is_built_in, created_by_user_id, created_at, updated_at, version)
		VALUES (@id, @name, @description, @notes, @is_built_in, @created_by_user_id, @created_at, @updated_at, @version)
	`, pgx.NamedArgs{
		"id":                 ex.ID,
		"name":               ex.Name,
		"description":        ex.Description,
		"notes":              ex.Notes,
		"is_built_in":        ex.IsBuiltIn,
		"created_by_user_id": ex.CreatedByUserID,
		"created_at":         ex.CreatedAt,
		"updated_at":         ex.UpdatedAt,
		"version":            ex.Version,
	}); err != nil {
		return domainexercise.Exercise{}, err
	}

	for _, m := range ex.Media {
		if _, err := conn.Exec(ctx, `
			INSERT INTO gym.exercise_media (id, exercise_id, media_type, sort_order, s3_key)
			VALUES (@id, @exercise_id, @media_type, @sort_order, @s3_key)
		`, pgx.NamedArgs{
			"id":          m.ID,
			argExerciseID: m.ExerciseID,
			"media_type":  m.MediaType,
			"sort_order":  m.SortOrder,
			"s3_key":      m.S3Key,
		}); err != nil {
			return domainexercise.Exercise{}, err
		}
	}

	for _, mg := range ex.MuscleGroups {
		if _, err := conn.Exec(ctx, `
			INSERT INTO gym.exercise_muscle_groups (exercise_id, muscle_group_id, is_primary)
			VALUES (@exercise_id, @muscle_group_id, @is_primary)
		`, pgx.NamedArgs{
			argExerciseID:     mg.ExerciseID,
			"muscle_group_id": mg.MuscleGroupID,
			"is_primary":      mg.IsPrimary,
		}); err != nil {
			return domainexercise.Exercise{}, err
		}
	}

	return ex, nil
}

func (r *ExerciseRepo) Update(ctx context.Context, ex domainexercise.Exercise) (domainexercise.Exercise, error) {
	conn := r.getter.DefaultTrOrDB(ctx, r.db)

	ex.Version++
	tag, err := conn.Exec(ctx, `
		UPDATE gym.exercises SET
			name = @name, description = @description, notes = @notes,
			updated_at = @updated_at, version = @version
		WHERE id = @id AND version = @expected_version
	`, pgx.NamedArgs{
		"name":             ex.Name,
		"description":      ex.Description,
		"notes":            ex.Notes,
		"updated_at":       ex.UpdatedAt,
		"version":          ex.Version,
		"id":               ex.ID,
		"expected_version": ex.Version - 1,
	})
	if err != nil {
		return domainexercise.Exercise{}, err
	}
	if tag.RowsAffected() == 0 {
		return domainexercise.Exercise{}, domainexercise.ErrExerciseNotFound
	}

	if _, err = conn.Exec(ctx, `
		DELETE FROM gym.exercise_media WHERE exercise_id = @exercise_id
	`, pgx.NamedArgs{argExerciseID: ex.ID}); err != nil {
		return domainexercise.Exercise{}, err
	}
	for _, m := range ex.Media {
		if _, err = conn.Exec(ctx, `
			INSERT INTO gym.exercise_media (id, exercise_id, media_type, sort_order, s3_key)
			VALUES (@id, @exercise_id, @media_type, @sort_order, @s3_key)
		`, pgx.NamedArgs{
			"id":          m.ID,
			argExerciseID: m.ExerciseID,
			"media_type":  m.MediaType,
			"sort_order":  m.SortOrder,
			"s3_key":      m.S3Key,
		}); err != nil {
			return domainexercise.Exercise{}, err
		}
	}

	if _, err = conn.Exec(ctx, `
		DELETE FROM gym.exercise_muscle_groups WHERE exercise_id = @exercise_id
	`, pgx.NamedArgs{argExerciseID: ex.ID}); err != nil {
		return domainexercise.Exercise{}, err
	}
	for _, mg := range ex.MuscleGroups {
		if _, err = conn.Exec(ctx, `
			INSERT INTO gym.exercise_muscle_groups (exercise_id, muscle_group_id, is_primary)
			VALUES (@exercise_id, @muscle_group_id, @is_primary)
		`, pgx.NamedArgs{
			argExerciseID:     mg.ExerciseID,
			"muscle_group_id": mg.MuscleGroupID,
			"is_primary":      mg.IsPrimary,
		}); err != nil {
			return domainexercise.Exercise{}, err
		}
	}

	return ex, nil
}

func (r *ExerciseRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	conn := r.getter.DefaultTrOrDB(ctx, r.db)
	_, err := conn.Exec(ctx, `
		UPDATE gym.exercises SET deleted_at = now(), updated_at = now(), version = version + 1
		WHERE id = @id
	`, pgx.NamedArgs{"id": id})
	if errors.Is(err, pgx.ErrNoRows) {
		return domainexercise.ErrExerciseNotFound
	}
	return err
}
