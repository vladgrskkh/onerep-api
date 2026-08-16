package template

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type MediaType string

const (
	MediaTypePhoto MediaType = "photo"
)

// AllowsContentType reports whether contentType is uploadable for the media
// type: photos accept JPEG, PNG, and WebP images.
func (m MediaType) AllowsContentType(contentType string) bool {
	if m != MediaTypePhoto {
		return false
	}
	switch contentType {
	case "image/jpeg", "image/png", "image/webp":
		return true
	}
	return false
}

type Template struct {
	ID              uuid.UUID
	Name            string
	Description     string
	IsPublic        bool
	CreatedByUserID uuid.UUID
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       time.Time
	Version         int
	Exercises       []TemplateExercise
	Media           []TemplateMedia
}

type TemplateMedia struct {
	ID         uuid.UUID
	TemplateID uuid.UUID
	MediaType  MediaType
	SortOrder  int
	S3Key      string
}

type TemplateExercise struct {
	TemplateID  uuid.UUID
	ExerciseID  uuid.UUID
	SortOrder   int
	PlannedSets int
}

// NextMediaSortOrder returns the highest existing media sort order plus one,
// or zero when the template has no media yet.
func NextMediaSortOrder(media []TemplateMedia) int {
	next := 0
	for _, m := range media {
		if m.SortOrder >= next {
			next = m.SortOrder + 1
		}
	}
	return next
}

func NewTemplate(name, description string, createdByUserID uuid.UUID) (Template, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Template{}, ErrInvalidName
	}
	if createdByUserID == uuid.Nil {
		return Template{}, ErrInvalidUserID
	}
	now := time.Now()
	return Template{
		ID:              uuid.Must(uuid.NewV7()),
		Name:            name,
		Description:     description,
		IsPublic:        false,
		CreatedByUserID: createdByUserID,
		CreatedAt:       now,
		UpdatedAt:       now,
		Version:         1,
	}, nil
}
