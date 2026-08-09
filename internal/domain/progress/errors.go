package progress

import "errors"

var (
	ErrInvalidUserID        = errors.New("user id must not be nil")
	ErrInvalidExerciseID    = errors.New("exercise id must not be nil")
	ErrInvalidMuscleGroupID = errors.New("muscle group id must be positive")
	ErrInvalidEstimated1RM  = errors.New("estimated 1rm must be greater than zero")
	ErrInvalidTotalKG       = errors.New("total kg must be greater than zero")
)
