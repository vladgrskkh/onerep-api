package template

import (
	"time"

	"github.com/google/uuid"
)

type TemplateFilter struct {
	UserID   uuid.UUID
	IsPublic *bool
	Since    time.Time
}
