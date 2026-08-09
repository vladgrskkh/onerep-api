package workout

import "github.com/google/uuid"

// StartWorkoutCommand carries the fields needed to start a workout. A nil
// TemplateID starts a workout without copying a template.
type StartWorkoutCommand struct {
	UserID     uuid.UUID
	TemplateID *uuid.UUID
}

// AddExerciseCommand carries the fields needed to append an exercise to a
// workout.
type AddExerciseCommand struct {
	WorkoutID  uuid.UUID
	ExerciseID uuid.UUID
}

// LogSetCommand carries the fields needed to record a set. WorkoutID is used
// for the ownership check; WorkoutExerciseID identifies the exercise within
// the workout.
type LogSetCommand struct {
	WorkoutID         uuid.UUID
	WorkoutExerciseID uuid.UUID
	WeightKg          float64
	Reps              int
	RPE               *int
	RestSeconds       *int
	IsWarmup          bool
}

// FinishWorkoutCommand carries the fields needed to complete a workout.
type FinishWorkoutCommand struct {
	WorkoutID uuid.UUID
}
