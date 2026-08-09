package exercise

import "errors"

var (
	ErrExerciseNotFound  = errors.New("exercise not found")
	ErrCannotEditBuiltIn = errors.New("cannot edit built-in exercises")
	ErrInvalidName       = errors.New("name must not be empty")
	ErrInvalidUserID     = errors.New("user id must not be nil")
	ErrInvalidID         = errors.New("id must be positive")
)
