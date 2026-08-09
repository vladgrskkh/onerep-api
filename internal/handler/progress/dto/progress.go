package dto

import (
	"time"

	"github.com/google/uuid"
)

type OneRMResponse struct {
	Date         time.Time `json:"date,omitzero"`
	Estimated1RM float64   `json:"estimated_1rm,omitzero"`
}

type VolumeResponse struct {
	Date          time.Time `json:"date,omitzero"`
	MuscleGroupID int       `json:"muscle_group_id,omitzero"`
	TotalKG       float64   `json:"total_kg,omitzero"`
}

type BodyWeightResponse struct {
	ID         uuid.UUID `json:"id,omitzero"`
	WeightKg   float64   `json:"weight_kg,omitzero"`
	MeasuredAt time.Time `json:"measured_at,omitzero"`
	CreatedAt  time.Time `json:"created_at,omitzero"`
}

type LogBodyWeightRequest struct {
	WeightKg   float64    `json:"weight_kg"            validate:"required,gt=0"`
	MeasuredAt *time.Time `json:"measured_at,omitzero"`
}
