package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"

	domainworkout "github.com/vladgrskkh/onerep-api/internal/domain/workout"
)

const (
	argFinishedAt = "finished_at"
	argNotes      = "notes"
)

type WorkoutRepo struct {
	db     trmpgx.Tr
	getter *trmpgx.CtxGetter
}

func NewWorkoutRepo(pool *pgxpool.Pool) *WorkoutRepo {
	return &WorkoutRepo{db: pool, getter: trmpgx.DefaultCtxGetter}
}

func (r *WorkoutRepo) List(
	ctx context.Context,
	filter domainworkout.WorkoutFilter,
) ([]*domainworkout.Workout, error) {
	query, args := buildWorkoutListQuery(filter)
	rows, err := r.getter.DefaultTrOrDB(ctx, r.db).Query(ctx, query, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workouts []*domainworkout.Workout
	for rows.Next() {
		w := &domainworkout.Workout{}
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

func buildWorkoutListQuery(filter domainworkout.WorkoutFilter) (string, pgx.NamedArgs) {
	var query strings.Builder
	query.WriteString(
		`SELECT id, user_id, template_id, started_at, finished_at, notes, created_at, updated_at, version FROM gym.workouts WHERE deleted_at IS NULL`,
	)
	args := pgx.NamedArgs{}

	if filter.UserID != nil {
		query.WriteString(` AND user_id = @user_id`)
		args["user_id"] = *filter.UserID
	}
	if !filter.Since.IsZero() {
		query.WriteString(` AND updated_at > @since`)
		args["since"] = filter.Since
	}
	query.WriteString(` ORDER BY started_at DESC`)

	return query.String(), args
}

func (r *WorkoutRepo) FindByID(ctx context.Context, id uuid.UUID) (*domainworkout.Workout, error) {
	conn := r.getter.DefaultTrOrDB(ctx, r.db)
	rows, err := conn.Query(ctx, `
		SELECT w.id, w.user_id, w.template_id, w.started_at, w.finished_at, w.notes,
		       w.created_at, w.updated_at, w.deleted_at, w.version,
		       we.id, we.exercise_id, we.sort_order, we.notes,
		       ws.id, ws.set_number, ws.weight_kg, ws.reps, ws.rpe, ws.rest_seconds, ws.is_warmup
		FROM gym.workouts w
		LEFT JOIN gym.workout_exercises we ON we.workout_id = w.id
		LEFT JOIN gym.workout_sets ws ON ws.workout_exercise_id = we.id
		WHERE w.id = @id AND w.deleted_at IS NULL
		ORDER BY we.sort_order, ws.set_number
	`, pgx.NamedArgs{"id": id})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		w           *domainworkout.Workout
		exerciseIdx = make(map[uuid.UUID]int)
	)
	for rows.Next() {
		if w == nil {
			w = &domainworkout.Workout{}
		}
		var (
			weID          *uuid.UUID
			weExerciseID  *uuid.UUID
			weSortOrder   *int
			weNotes       *string
			wsID          *uuid.UUID
			wsSetNumber   *int
			wsWeightKg    *float64
			wsReps        *int
			wsRPE         *int
			wsRestSeconds *int
			wsIsWarmup    *bool
		)
		if scanErr := rows.Scan(
			&w.ID, &w.UserID, &w.TemplateID, &w.StartedAt, &w.FinishedAt, &w.Notes,
			&w.CreatedAt, &w.UpdatedAt, &w.DeletedAt, &w.Version,
			&weID, &weExerciseID, &weSortOrder, &weNotes,
			&wsID, &wsSetNumber, &wsWeightKg, &wsReps, &wsRPE, &wsRestSeconds, &wsIsWarmup,
		); scanErr != nil {
			return nil, scanErr
		}
		if weID != nil {
			if _, seen := exerciseIdx[*weID]; !seen {
				idx := len(w.Exercises)
				w.Exercises = append(w.Exercises, domainworkout.WorkoutExercise{
					ID:         *weID,
					WorkoutID:  w.ID,
					ExerciseID: *weExerciseID,
					SortOrder:  *weSortOrder,
					Notes:      *weNotes,
				})
				exerciseIdx[*weID] = idx
			}
		}
		if wsID != nil {
			if idx, ok := exerciseIdx[*weID]; ok {
				w.Exercises[idx].Sets = append(w.Exercises[idx].Sets, domainworkout.WorkoutSet{
					ID:                *wsID,
					WorkoutExerciseID: *weID,
					SetNumber:         *wsSetNumber,
					WeightKg:          *wsWeightKg,
					Reps:              *wsReps,
					RPE:               wsRPE,
					RestSeconds:       wsRestSeconds,
					IsWarmup:          *wsIsWarmup,
				})
			}
		}
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, rowsErr
	}
	if w == nil {
		return nil, domainworkout.ErrWorkoutNotFound
	}
	return w, nil
}

func (r *WorkoutRepo) Create(ctx context.Context, w domainworkout.Workout) (*domainworkout.Workout, error) {
	conn := r.getter.DefaultTrOrDB(ctx, r.db)
	if _, err := conn.Exec(ctx, `
		INSERT INTO gym.workouts (id, user_id, template_id, started_at, finished_at, notes, created_at, updated_at, version)
		VALUES (@id, @user_id, @template_id, @started_at, @finished_at, @notes, @created_at, @updated_at, @version)
	`, pgx.NamedArgs{
		"id":          w.ID,
		"user_id":     w.UserID,
		"template_id": w.TemplateID,
		"started_at":  w.StartedAt,
		argFinishedAt: w.FinishedAt,
		argNotes:      w.Notes,
		"created_at":  w.CreatedAt,
		"updated_at":  w.UpdatedAt,
		"version":     w.Version,
	}); err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *WorkoutRepo) AddExercise(
	ctx context.Context,
	workoutID, exerciseID uuid.UUID,
) (*domainworkout.WorkoutExercise, error) {
	we, err := domainworkout.NewWorkoutExercise(workoutID, exerciseID)
	if err != nil {
		return nil, err
	}

	conn := r.getter.DefaultTrOrDB(ctx, r.db)
	err = conn.QueryRow(ctx, `
		INSERT INTO gym.workout_exercises (id, workout_id, exercise_id, sort_order, notes)
		VALUES (@id, @workout_id, @exercise_id,
		        (SELECT COALESCE(MAX(sort_order), -1) + 1 FROM gym.workout_exercises WHERE workout_id = @workout_id), '')
		RETURNING sort_order
	`, pgx.NamedArgs{
		"id":          we.ID,
		"workout_id":  we.WorkoutID,
		"exercise_id": we.ExerciseID,
	}).Scan(&we.SortOrder)
	if err != nil {
		return nil, err
	}
	return &we, nil
}

func (r *WorkoutRepo) LogSet(
	ctx context.Context,
	workoutExerciseID uuid.UUID,
	set domainworkout.WorkoutSet,
) (*domainworkout.WorkoutSet, error) {
	set.WorkoutExerciseID = workoutExerciseID
	conn := r.getter.DefaultTrOrDB(ctx, r.db)
	err := conn.QueryRow(ctx, `
		INSERT INTO gym.workout_sets (id, workout_exercise_id, set_number, weight_kg, reps, rpe, rest_seconds, is_warmup)
		VALUES (@id, @workout_exercise_id,
		        (SELECT COALESCE(MAX(set_number), 0) + 1 FROM gym.workout_sets WHERE workout_exercise_id = @workout_exercise_id),
		        @weight_kg, @reps, @rpe, @rest_seconds, @is_warmup)
		RETURNING set_number
	`, pgx.NamedArgs{
		"id":                  set.ID,
		"workout_exercise_id": set.WorkoutExerciseID,
		"weight_kg":           set.WeightKg,
		"reps":                set.Reps,
		"rpe":                 set.RPE,
		"rest_seconds":        set.RestSeconds,
		"is_warmup":           set.IsWarmup,
	}).Scan(&set.SetNumber)
	if err != nil {
		return nil, err
	}
	return &set, nil
}

func (r *WorkoutRepo) BatchInsertExercises(
	ctx context.Context,
	exercises []domainworkout.WorkoutExercise,
) error {
	if len(exercises) == 0 {
		return nil
	}
	conn := r.getter.DefaultTrOrDB(ctx, r.db)
	batch := &pgx.Batch{}
	for _, we := range exercises {
		batch.Queue(`
			INSERT INTO gym.workout_exercises (id, workout_id, exercise_id, sort_order, notes)
			VALUES (@id, @workout_id, @exercise_id, @sort_order, @notes)
		`, pgx.NamedArgs{
			"id":          we.ID,
			"workout_id":  we.WorkoutID,
			"exercise_id": we.ExerciseID,
			"sort_order":  we.SortOrder,
			argNotes:      we.Notes,
		})
	}
	return conn.SendBatch(ctx, batch).Close()
}

func (r *WorkoutRepo) BatchInsertSets(ctx context.Context, sets []domainworkout.WorkoutSet) error {
	if len(sets) == 0 {
		return nil
	}
	conn := r.getter.DefaultTrOrDB(ctx, r.db)
	batch := &pgx.Batch{}
	for _, set := range sets {
		batch.Queue(`
			INSERT INTO gym.workout_sets (id, workout_exercise_id, set_number, weight_kg, reps, rpe, rest_seconds, is_warmup)
			VALUES (@id, @workout_exercise_id, @set_number, @weight_kg, @reps, @rpe, @rest_seconds, @is_warmup)
		`, pgx.NamedArgs{
			"id":                  set.ID,
			"workout_exercise_id": set.WorkoutExerciseID,
			"set_number":          set.SetNumber,
			"weight_kg":           set.WeightKg,
			"reps":                set.Reps,
			"rpe":                 set.RPE,
			"rest_seconds":        set.RestSeconds,
			"is_warmup":           set.IsWarmup,
		})
	}
	return conn.SendBatch(ctx, batch).Close()
}

func (r *WorkoutRepo) Update(ctx context.Context, w domainworkout.Workout) (*domainworkout.Workout, error) {
	conn := r.getter.DefaultTrOrDB(ctx, r.db)

	w.Version++
	tag, err := conn.Exec(ctx, `
		UPDATE gym.workouts SET
			notes = @notes, finished_at = @finished_at,
			updated_at = @updated_at, version = @version
		WHERE id = @id AND version = @expected_version
	`, pgx.NamedArgs{
		argNotes:           w.Notes,
		argFinishedAt:      w.FinishedAt,
		"updated_at":       w.UpdatedAt,
		"version":          w.Version,
		"id":               w.ID,
		"expected_version": w.Version - 1,
	})
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, domainworkout.ErrWorkoutNotFound
	}
	return &w, nil
}

func (r *WorkoutRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	conn := r.getter.DefaultTrOrDB(ctx, r.db)
	_, err := conn.Exec(ctx, `
		UPDATE gym.workouts SET deleted_at = now(), updated_at = now(), version = version + 1
		WHERE id = @id
	`, pgx.NamedArgs{"id": id})
	return err
}

func (r *WorkoutRepo) Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (*domainworkout.Workout, error) {
	conn := r.getter.DefaultTrOrDB(ctx, r.db)
	tag, err := conn.Exec(ctx, `
		UPDATE gym.workouts SET finished_at = @finished_at, updated_at = now(), version = version + 1
		WHERE id = @id AND deleted_at IS NULL
	`, pgx.NamedArgs{argFinishedAt: finishedAt, "id": id})
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, domainworkout.ErrWorkoutNotFound
	}
	return r.FindByID(ctx, id)
}
