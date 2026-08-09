package template

import "errors"

var (
	ErrTemplateNotFound = errors.New("template not found")
	ErrNotOwner         = errors.New("not the owner of this resource")
)
