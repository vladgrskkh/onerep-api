package exercise

import "errors"

var (
	ErrExerciseNotFound  = errors.New("exercise not found")
	ErrCannotEditBuiltIn = errors.New("cannot edit built-in exercises")
)
