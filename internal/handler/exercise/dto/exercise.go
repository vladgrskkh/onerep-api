package dto

import (
	"time"

	"github.com/google/uuid"
)

type ExerciseCreateRequest struct {
	Name           string `json:"name"                      validate:"required,max=100"`
	Description    string `json:"description,omitzero"`
	Notes          string `json:"notes,omitzero"`
	MuscleGroupIDs []int  `json:"muscle_group_ids,omitzero" validate:"omitempty,dive,gt=0"`
}

type ExerciseUpdateRequest struct {
	Name           *string `json:"name,omitzero"             validate:"omitempty,max=100"`
	Description    *string `json:"description,omitzero"`
	Notes          *string `json:"notes,omitzero"`
	MuscleGroupIDs *[]int  `json:"muscle_group_ids,omitzero" validate:"omitempty,dive,gt=0"`
}

type ExerciseResponse struct {
	ID              uuid.UUID               `json:"id"`
	Name            string                  `json:"name"`
	Description     string                  `json:"description,omitzero"`
	Notes           string                  `json:"notes,omitzero"`
	IsBuiltIn       bool                    `json:"is_built_in"`
	CreatedByUserID uuid.UUID               `json:"created_by_user_id,omitzero"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
	Version         int                     `json:"version"`
	Media           []ExerciseMediaResponse `json:"media,omitzero"`
	MuscleGroups    []MuscleGroupResponse   `json:"muscle_groups,omitzero"`
}

type ExerciseMediaResponse struct {
	ID        uuid.UUID `json:"id"`
	MediaType string    `json:"media_type"`
	SortOrder int       `json:"sort_order"`
	S3Key     string    `json:"s3_key"`
}

type MuscleGroupResponse struct {
	ID        int  `json:"id"`
	IsPrimary bool `json:"is_primary"`
}
