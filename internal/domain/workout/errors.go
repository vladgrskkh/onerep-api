package workout

import "errors"

var (
	ErrWorkoutNotFound = errors.New("workout not found")
	ErrActiveWorkout   = errors.New("user already has an active workout")
)
