package dto

import (
	"time"

	"github.com/google/uuid"
)

type TemplateCreateRequest struct {
	Name        string                 `json:"name"                 validate:"required,max=100"`
	Description string                 `json:"description,omitzero"`
	Exercises   []TemplateExerciseItem `json:"exercises,omitzero"   validate:"omitempty,dive"`
}

type TemplateUpdateRequest struct {
	Name        *string                 `json:"name,omitzero"        validate:"omitempty,max=100"`
	Description *string                 `json:"description,omitzero"`
	Exercises   *[]TemplateExerciseItem `json:"exercises,omitzero"   validate:"omitempty,dive"`
}

type TemplateExerciseItem struct {
	ExerciseID  uuid.UUID `json:"exercise_id"           validate:"required"`
	PlannedSets int       `json:"planned_sets,omitzero" validate:"omitempty,gt=0"`
}

type TemplateResponse struct {
	ID              uuid.UUID                  `json:"id"`
	Name            string                     `json:"name"`
	Description     string                     `json:"description,omitzero"`
	IsPublic        bool                       `json:"is_public"`
	CreatedByUserID uuid.UUID                  `json:"created_by_user_id"`
	CreatedAt       time.Time                  `json:"created_at"`
	UpdatedAt       time.Time                  `json:"updated_at"`
	Version         int                        `json:"version"`
	Exercises       []TemplateExerciseResponse `json:"exercises,omitzero"`
	Media           []TemplateMediaResponse    `json:"media,omitzero"`
}

type TemplateExerciseResponse struct {
	ExerciseID  uuid.UUID `json:"exercise_id"`
	SortOrder   int       `json:"sort_order"`
	PlannedSets int       `json:"planned_sets"`
}

type TemplateMediaResponse struct {
	ID        uuid.UUID `json:"id"`
	MediaType string    `json:"media_type"`
	SortOrder int       `json:"sort_order"`
	S3Key     string    `json:"s3_key"`
}
