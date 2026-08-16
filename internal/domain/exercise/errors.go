package exercise

import "errors"

var (
	ErrExerciseNotFound       = errors.New("exercise not found")
	ErrCannotEditBuiltIn      = errors.New("cannot edit built-in exercises")
	ErrMuscleGroupMissing     = errors.New("one or more muscle groups do not exist")
	ErrInvalidName            = errors.New("name must not be empty")
	ErrInvalidUserID          = errors.New("user id must not be nil")
	ErrInvalidID              = errors.New("id must be positive")
	ErrInvalidMediaType       = errors.New("media type must be photo or video")
	ErrUnsupportedContentType = errors.New("content type is not allowed for this media type")
)
