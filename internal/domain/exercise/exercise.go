package exercise

import (
	"time"

	"github.com/google/uuid"
)

type MediaType string

const (
	MediaTypePhoto MediaType = "photo"
	MediaTypeVideo MediaType = "video"
)

type Exercise struct {
	ID              uuid.UUID
	Name            string
	Description     string
	Notes           string
	IsBuiltIn       bool
	CreatedByUserID *uuid.UUID
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
	Version         int
	Media           []ExerciseMedia
	MuscleGroups    []ExerciseMuscleGroup
}

type ExerciseMedia struct {
	ID         uuid.UUID
	ExerciseID uuid.UUID
	MediaType  MediaType
	SortOrder  int
	S3Key      string
}

type ExerciseMuscleGroup struct {
	ExerciseID    uuid.UUID
	MuscleGroupID int
	IsPrimary     bool
}

func NewExercise(name, description, notes string, createdByUserID uuid.UUID) Exercise {
	now := time.Now()
	return Exercise{
		ID:              uuid.Must(uuid.NewV7()),
		Name:            name,
		Description:     description,
		Notes:           notes,
		IsBuiltIn:       false,
		CreatedByUserID: &createdByUserID,
		CreatedAt:       now,
		UpdatedAt:       now,
		Version:         1,
	}
}
