package workout

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainprogress "github.com/vladgrskkh/onerep-api/internal/domain/progress"
	domainworkout "github.com/vladgrskkh/onerep-api/internal/domain/workout"
	serviceexercise "github.com/vladgrskkh/onerep-api/internal/service/exercise"
	serviceprogress "github.com/vladgrskkh/onerep-api/internal/service/progress"
)

// day truncates workout started_at to its UTC day.
const day = 24 * time.Hour

// VolumeCalculator recomputes the user's per-muscle-group volume for a
// finished workout. Every non-warm-up set contributes weight_kg × reps to
// each muscle group linked to its exercise (primary and secondary alike), and
// the workout's totals are stored under the UTC day its started_at falls on.
type VolumeCalculator struct {
	workouts  WorkoutRepository
	exercises serviceexercise.ExerciseRepository
	progress  serviceprogress.ProgressRepository
	trManager TransactionManager
}

// NewVolumeCalculator creates a calculator reading workouts and exercises and
// writing progress volume.
func NewVolumeCalculator(
	workouts WorkoutRepository,
	exercises serviceexercise.ExerciseRepository,
	progress serviceprogress.ProgressRepository,
	trManager TransactionManager,
) *VolumeCalculator {
	return &VolumeCalculator{
		workouts:  workouts,
		exercises: exercises,
		progress:  progress,
		trManager: trManager,
	}
}

// Recalculate loads the workout, aggregates its set volume by muscle group,
// and upserts one progress volume row per group. The upserts run in a single
// transaction so that a mid-loop failure rolls back every group and the
// stored totals stay consistent.
func (c *VolumeCalculator) Recalculate(ctx context.Context, workoutID, userID uuid.UUID) error {
	workout, err := c.workouts.FindByID(ctx, workoutID)
	if err != nil {
		return err
	}
	if workout.UserID != userID {
		return domainworkout.ErrWorkoutNotFound
	}

	totals := make(map[int]float64)
	for _, we := range workout.Exercises {
		ex, err := c.exercises.FindByID(ctx, we.ExerciseID)
		if err != nil {
			return err
		}
		volume := setVolume(we)
		for _, mg := range ex.MuscleGroups {
			totals[mg.MuscleGroupID] += volume
		}
	}

	date := workout.StartedAt.UTC().Truncate(day)
	return c.trManager.Do(ctx, func(ctx context.Context) error {
		for muscleGroupID, total := range totals {
			p, err := domainprogress.NewProgressVolume(muscleGroupID, userID, date, total)
			if err != nil {
				return err
			}
			if err := c.progress.UpsertVolume(ctx, p); err != nil {
				return err
			}
		}
		return nil
	})
}

// setVolume returns the total tonnage of the exercise's non-warm-up sets.
func setVolume(we domainworkout.WorkoutExercise) float64 {
	var volume float64
	for _, set := range we.Sets {
		if set.IsWarmup {
			continue
		}
		volume += set.WeightKg * float64(set.Reps)
	}
	return volume
}
