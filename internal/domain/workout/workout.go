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

func NewWorkout(userID uuid.UUID, templateID *uuid.UUID) Workout {
	now := time.Now()
	return Workout{
		ID:         uuid.Must(uuid.NewV7()),
		UserID:     userID,
		TemplateID: templateID,
		StartedAt:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
		Version:    1,
	}
}

func NewWorkoutExercise(workoutID, exerciseID uuid.UUID) WorkoutExercise {
	return WorkoutExercise{
		ID:         uuid.Must(uuid.NewV7()),
		WorkoutID:  workoutID,
		ExerciseID: exerciseID,
	}
}

func NewWorkoutSet(workoutExerciseID uuid.UUID, weightKg float64, reps int, rpe *int, restSeconds *int, isWarmup bool) WorkoutSet {
	return WorkoutSet{
		ID:                uuid.Must(uuid.NewV7()),
		WorkoutExerciseID: workoutExerciseID,
		WeightKg:          weightKg,
		Reps:              reps,
		RPE:               rpe,
		RestSeconds:       restSeconds,
		IsWarmup:          isWarmup,
	}
}
