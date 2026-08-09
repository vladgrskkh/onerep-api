package postgres

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domainworkout "github.com/vladgrskkh/onerep-api/internal/domain/workout"
)

type WorkoutRepo struct {
	pool *pgxpool.Pool
}

func NewWorkoutRepo(pool *pgxpool.Pool) *WorkoutRepo {
	return &WorkoutRepo{pool: pool}
}

func (r *WorkoutRepo) List(
	ctx context.Context,
	filter domainworkout.WorkoutFilter,
) ([]domainworkout.Workout, error) {
	query, args := buildWorkoutListQuery(filter)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workouts []domainworkout.Workout
	for rows.Next() {
		var w domainworkout.Workout
		if scanErr := rows.Scan(
			&w.ID,
			&w.UserID,
			&w.TemplateID,
			&w.StartedAt,
			&w.FinishedAt,
			&w.Notes,
			&w.CreatedAt,
			&w.UpdatedAt,
			&w.Version,
		); scanErr != nil {
			return nil, scanErr
		}
		workouts = append(workouts, w)
	}
	return workouts, rows.Err()
}

func buildWorkoutListQuery(filter domainworkout.WorkoutFilter) (string, []any) {
	var query strings.Builder
	query.WriteString(
		`SELECT id, user_id, template_id, started_at, finished_at, notes, created_at, updated_at, version FROM gym.workouts WHERE deleted_at IS NULL`,
	)
	var args []any

	if filter.UserID != nil {
		args = append(args, *filter.UserID)
		query.WriteString(" AND user_id = $" + strconv.Itoa(len(args)))
	}
	if filter.Since != nil {
		args = append(args, *filter.Since)
		query.WriteString(" AND updated_at > $" + strconv.Itoa(len(args)))
	}
	query.WriteString(" ORDER BY started_at DESC")

	return query.String(), args
}

func (r *WorkoutRepo) FindByID(ctx context.Context, id uuid.UUID) (domainworkout.Workout, error) {
	var w domainworkout.Workout
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, template_id, started_at, finished_at, notes, created_at, updated_at, deleted_at, version
		FROM gym.workouts WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&w.ID, &w.UserID, &w.TemplateID, &w.StartedAt, &w.FinishedAt, &w.Notes, &w.CreatedAt, &w.UpdatedAt, &w.DeletedAt, &w.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return domainworkout.Workout{}, domainworkout.ErrWorkoutNotFound
	}
	if err != nil {
		return domainworkout.Workout{}, err
	}

	w.Exercises, err = r.findExercises(ctx, w.ID)
	if err != nil {
		return domainworkout.Workout{}, err
	}
	for i := range w.Exercises {
		w.Exercises[i].Sets, err = r.findSets(ctx, w.Exercises[i].ID)
		if err != nil {
			return domainworkout.Workout{}, err
		}
	}
	return w, nil
}

func (r *WorkoutRepo) findExercises(
	ctx context.Context,
	workoutID uuid.UUID,
) ([]domainworkout.WorkoutExercise, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, workout_id, exercise_id, sort_order, notes
		FROM gym.workout_exercises WHERE workout_id = $1 ORDER BY sort_order
	`, workoutID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exercises []domainworkout.WorkoutExercise
	for rows.Next() {
		var we domainworkout.WorkoutExercise
		if scanErr := rows.Scan(&we.ID, &we.WorkoutID, &we.ExerciseID, &we.SortOrder, &we.Notes); scanErr != nil {
			return nil, scanErr
		}
		exercises = append(exercises, we)
	}
	return exercises, rows.Err()
}

func (r *WorkoutRepo) findSets(
	ctx context.Context,
	workoutExerciseID uuid.UUID,
) ([]domainworkout.WorkoutSet, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, workout_exercise_id, set_number, weight_kg, reps, rpe, rest_seconds, is_warmup
		FROM gym.workout_sets WHERE workout_exercise_id = $1 ORDER BY set_number
	`, workoutExerciseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sets []domainworkout.WorkoutSet
	for rows.Next() {
		var s domainworkout.WorkoutSet
		if scanErr := rows.Scan(
			&s.ID,
			&s.WorkoutExerciseID,
			&s.SetNumber,
			&s.WeightKg,
			&s.Reps,
			&s.RPE,
			&s.RestSeconds,
			&s.IsWarmup,
		); scanErr != nil {
			return nil, scanErr
		}
		sets = append(sets, s)
	}
	return sets, rows.Err()
}

func (r *WorkoutRepo) Create(ctx context.Context, w domainworkout.Workout) (domainworkout.Workout, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domainworkout.Workout{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err = tx.Exec(ctx, `
		INSERT INTO gym.workouts (id, user_id, template_id, started_at, finished_at, notes, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, w.ID, w.UserID, w.TemplateID, w.StartedAt, w.FinishedAt, w.Notes, w.CreatedAt, w.UpdatedAt, w.Version); err != nil {
		return domainworkout.Workout{}, err
	}

	for _, we := range w.Exercises {
		if _, err = tx.Exec(ctx, `
			INSERT INTO gym.workout_exercises (id, workout_id, exercise_id, sort_order, notes)
			VALUES ($1, $2, $3, $4, $5)
		`, we.ID, we.WorkoutID, we.ExerciseID, we.SortOrder, we.Notes); err != nil {
			return domainworkout.Workout{}, err
		}
		for _, s := range we.Sets {
			if _, err = tx.Exec(ctx, `
				INSERT INTO gym.workout_sets (id, workout_exercise_id, set_number, weight_kg, reps, rpe, rest_seconds, is_warmup)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			`, s.ID, s.WorkoutExerciseID, s.SetNumber, s.WeightKg, s.Reps, s.RPE, s.RestSeconds, s.IsWarmup); err != nil {
				return domainworkout.Workout{}, err
			}
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return domainworkout.Workout{}, err
	}
	return w, nil
}

func (r *WorkoutRepo) AddExercise(
	ctx context.Context,
	workoutID, exerciseID uuid.UUID,
) (domainworkout.WorkoutExercise, error) {
	we, err := domainworkout.NewWorkoutExercise(workoutID, exerciseID)
	if err != nil {
		return domainworkout.WorkoutExercise{}, err
	}

	err = r.pool.QueryRow(ctx, `
		INSERT INTO gym.workout_exercises (id, workout_id, exercise_id, sort_order, notes)
		VALUES ($1, $2, $3, (SELECT COALESCE(MAX(sort_order), -1) + 1 FROM gym.workout_exercises WHERE workout_id = $2), '')
		RETURNING sort_order
	`, we.ID, we.WorkoutID, we.ExerciseID).Scan(&we.SortOrder)
	if err != nil {
		return domainworkout.WorkoutExercise{}, err
	}
	return we, nil
}

func (r *WorkoutRepo) LogSet(
	ctx context.Context,
	workoutExerciseID uuid.UUID,
	set domainworkout.WorkoutSet,
) (domainworkout.WorkoutSet, error) {
	set.WorkoutExerciseID = workoutExerciseID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO gym.workout_sets (id, workout_exercise_id, set_number, weight_kg, reps, rpe, rest_seconds, is_warmup)
		VALUES ($1, $2, (SELECT COALESCE(MAX(set_number), 0) + 1 FROM gym.workout_sets WHERE workout_exercise_id = $2), $3, $4, $5, $6, $7)
		RETURNING set_number
	`, set.ID, set.WorkoutExerciseID, set.WeightKg, set.Reps, set.RPE, set.RestSeconds, set.IsWarmup).Scan(&set.SetNumber)
	if err != nil {
		return domainworkout.WorkoutSet{}, err
	}
	return set, nil
}

func (r *WorkoutRepo) Update(ctx context.Context, w domainworkout.Workout) (domainworkout.Workout, error) {
	w.Version++
	tag, err := r.pool.Exec(ctx, `
		UPDATE gym.workouts SET
			notes = $1, finished_at = $2,
			updated_at = $3, version = $4
		WHERE id = $5 AND version = $6
	`, w.Notes, w.FinishedAt, w.UpdatedAt, w.Version, w.ID, w.Version-1)
	if err != nil {
		return domainworkout.Workout{}, err
	}
	if tag.RowsAffected() == 0 {
		return domainworkout.Workout{}, domainworkout.ErrWorkoutNotFound
	}
	return w, nil
}

func (r *WorkoutRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE gym.workouts SET deleted_at = now(), updated_at = now(), version = version + 1
		WHERE id = $1
	`, id)
	return err
}

func (r *WorkoutRepo) Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (domainworkout.Workout, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE gym.workouts SET finished_at = $2, updated_at = now(), version = version + 1
		WHERE id = $1 AND deleted_at IS NULL
	`, id, finishedAt)
	if err != nil {
		return domainworkout.Workout{}, err
	}
	if tag.RowsAffected() == 0 {
		return domainworkout.Workout{}, domainworkout.ErrWorkoutNotFound
	}
	return r.FindByID(ctx, id)
}
