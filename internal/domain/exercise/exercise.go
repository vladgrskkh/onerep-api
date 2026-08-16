package exercise

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type MediaType string

const (
	MediaTypePhoto MediaType = "photo"
	MediaTypeVideo MediaType = "video"
)

// AllowsContentType reports whether contentType is uploadable for the media
// type: photos accept JPEG, PNG, and WebP images; videos accept MP4.
func (m MediaType) AllowsContentType(contentType string) bool {
	switch m {
	case MediaTypePhoto:
		switch contentType {
		case "image/jpeg", "image/png", "image/webp":
			return true
		}
		return false
	case MediaTypeVideo:
		return contentType == "video/mp4"
	}
	return false
}

type Exercise struct {
	ID              uuid.UUID
	Name            string
	Description     string
	Notes           string
	IsBuiltIn       bool
	CreatedByUserID uuid.UUID
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       time.Time
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

// NextMediaSortOrder returns the highest existing media sort order plus one,
// or zero when the exercise has no media yet.
func NextMediaSortOrder(media []ExerciseMedia) int {
	next := 0
	for _, m := range media {
		if m.SortOrder >= next {
			next = m.SortOrder + 1
		}
	}
	return next
}

// NewExercise creates a user exercise. A nil createdByUserID marks a
// built-in exercise.
func NewExercise(name, description, notes string, createdByUserID uuid.UUID) (Exercise, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Exercise{}, ErrInvalidName
	}
	now := time.Now()
	return Exercise{
		ID:              uuid.Must(uuid.NewV7()),
		Name:            name,
		Description:     description,
		Notes:           notes,
		IsBuiltIn:       false,
		CreatedByUserID: createdByUserID,
		CreatedAt:       now,
		UpdatedAt:       now,
		Version:         1,
	}, nil
}
