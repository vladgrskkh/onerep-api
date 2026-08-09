package bodyweight

import "errors"

var (
	ErrInvalidUserID = errors.New("user id must not be nil")
	ErrInvalidWeight = errors.New("weight must be greater than zero")
)
