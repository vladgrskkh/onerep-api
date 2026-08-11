package template

import "errors"

var (
	ErrTemplateNotFound = errors.New("template not found")
	ErrNotOwner         = errors.New("not the owner of this resource")
	ErrInvalidName      = errors.New("name must not be empty")
	ErrInvalidUserID    = errors.New("user id must not be nil")
	ErrInvalidS3Key     = errors.New("s3 key must not be empty")
)
