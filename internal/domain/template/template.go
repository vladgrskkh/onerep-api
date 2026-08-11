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
