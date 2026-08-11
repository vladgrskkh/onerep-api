package workout

import "errors"

var (
	ErrWorkoutNotFound          = errors.New("workout not found")
	ErrWorkoutExerciseNotFound  = errors.New("workout exercise not found")
	ErrActiveWorkout            = errors.New("user already has an active workout")
	ErrInvalidUserID            = errors.New("user id must not be nil")
	ErrInvalidWorkoutID         = errors.New("workout id must not be nil")
	ErrInvalidExerciseID        = errors.New("exercise id must not be nil")
	ErrInvalidWorkoutExerciseID = errors.New("workout exercise id must not be nil")
	ErrInvalidWeight            = errors.New("weight must be greater than zero")
	ErrInvalidReps              = errors.New("reps must be greater than zero")
	ErrInvalidRPE               = errors.New("rpe must be between 1 and 10")
	ErrInvalidRestSeconds       = errors.New("rest seconds must not be negative")
)
