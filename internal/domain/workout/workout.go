package workout

import (
	"time"

	"github.com/google/uuid"
)

type Workout struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TemplateID *uuid.UUID
	StartedAt  time.Time
	FinishedAt *time.Time
	Notes      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
	Version    int
	Exercises  []WorkoutExercise
}

type WorkoutExercise struct {
	ID         uuid.UUID
	WorkoutID  uuid.UUID
	ExerciseID uuid.UUID
	SortOrder  int
	Notes      string
	Sets       []WorkoutSet
}

type WorkoutSet struct {
	ID                uuid.UUID
	WorkoutExerciseID uuid.UUID
	SetNumber         int
	WeightKg          float64
	Reps              int
	RPE               *int
	RestSeconds       *int
	IsWarmup          bool
}

// SetResult reports a logged set together with its PR status and the
// estimated one-rep max it achieved.
type SetResult struct {
	WorkoutSet

	IsPR         bool
	Estimated1RM float64
}

func NewWorkout(userID, templateID uuid.UUID) (Workout, error) {
	if userID == uuid.Nil {
		return Workout{}, ErrInvalidUserID
	}
	var templateIDPtr *uuid.UUID
	if templateID != uuid.Nil {
		templateIDPtr = &templateID
	}
	now := time.Now()
	return Workout{
		ID:         uuid.Must(uuid.NewV7()),
		UserID:     userID,
		TemplateID: templateIDPtr,
		StartedAt:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
		Version:    1,
	}, nil
}

func NewWorkoutExercise(workoutID, exerciseID uuid.UUID) (WorkoutExercise, error) {
	if workoutID == uuid.Nil {
		return WorkoutExercise{}, ErrInvalidWorkoutID
	}
	if exerciseID == uuid.Nil {
		return WorkoutExercise{}, ErrInvalidExerciseID
	}
	return WorkoutExercise{
		ID:         uuid.Must(uuid.NewV7()),
		WorkoutID:  workoutID,
		ExerciseID: exerciseID,
	}, nil
}

func NewWorkoutSet(
	workoutExerciseID uuid.UUID,
	weightKg float64,
	reps int,
	rpe *int,
	restSeconds *int,
	isWarmup bool,
) (WorkoutSet, error) {
	if workoutExerciseID == uuid.Nil {
		return WorkoutSet{}, ErrInvalidWorkoutExerciseID
	}
	if weightKg <= 0 {
		return WorkoutSet{}, ErrInvalidWeight
	}
	if reps <= 0 {
		return WorkoutSet{}, ErrInvalidReps
	}
	if rpe != nil && (*rpe < 1 || *rpe > 10) {
		return WorkoutSet{}, ErrInvalidRPE
	}
	if restSeconds != nil && *restSeconds < 0 {
		return WorkoutSet{}, ErrInvalidRestSeconds
	}
	return WorkoutSet{
		ID:                uuid.Must(uuid.NewV7()),
		WorkoutExerciseID: workoutExerciseID,
		WeightKg:          weightKg,
		Reps:              reps,
		RPE:               rpe,
		RestSeconds:       restSeconds,
		IsWarmup:          isWarmup,
	}, nil
}
