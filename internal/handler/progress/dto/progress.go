package dto

import (
	"time"

	"github.com/google/uuid"
)

type OneRMResponse struct {
	Date         time.Time `json:"date"`
	Estimated1RM float64   `json:"estimated_1rm"`
}

type VolumeResponse struct {
	Date          time.Time `json:"date"`
	MuscleGroupID int       `json:"muscle_group_id"`
	TotalKG       float64   `json:"total_kg"`
}

type BodyWeightResponse struct {
	ID         uuid.UUID `json:"id"`
	WeightKg   float64   `json:"weight_kg"`
	MeasuredAt time.Time `json:"measured_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type LogBodyWeightRequest struct {
	WeightKg   float64   `json:"weight_kg"            validate:"required,gt=0"`
	MeasuredAt time.Time `json:"measured_at,omitzero"`
}
