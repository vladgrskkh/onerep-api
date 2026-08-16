package template

import "errors"

var (
	ErrTemplateNotFound       = errors.New("template not found")
	ErrNotOwner               = errors.New("not the owner of this resource")
	ErrInvalidName            = errors.New("name must not be empty")
	ErrInvalidUserID          = errors.New("user id must not be nil")
	ErrUnsupportedContentType = errors.New("content type is not allowed for template media")
)
