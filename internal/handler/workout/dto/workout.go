package dto

import (
	"time"

	"github.com/google/uuid"
)

type StartWorkoutRequest struct {
	TemplateID *uuid.UUID `json:"template_id,omitzero"`
}

type AddExerciseRequest struct {
	ExerciseID uuid.UUID `json:"exercise_id" validate:"required"`
}

type LogSetRequest struct {
	WeightKg    float64 `json:"weight_kg"             validate:"required,gt=0"`
	Reps        int     `json:"reps"                  validate:"required,gt=0"`
	RPE         *int    `json:"rpe,omitzero"          validate:"omitempty,min=1,max=10"`
	RestSeconds *int    `json:"rest_seconds,omitzero" validate:"omitempty,min=0"`
	IsWarmup    bool    `json:"is_warmup,omitzero"`
}

type WorkoutResponse struct {
	ID         uuid.UUID                 `json:"id"`
	UserID     uuid.UUID                 `json:"user_id,omitzero"`
	TemplateID *uuid.UUID                `json:"template_id,omitzero"`
	StartedAt  time.Time                 `json:"started_at"`
	FinishedAt *time.Time                `json:"finished_at,omitzero"`
	Notes      string                    `json:"notes,omitzero"`
	Exercises  []WorkoutExerciseResponse `json:"exercises,omitzero"`
	CreatedAt  time.Time                 `json:"created_at"`
	UpdatedAt  time.Time                 `json:"updated_at"`
}

type WorkoutExerciseResponse struct {
	ID         uuid.UUID            `json:"id"`
	ExerciseID uuid.UUID            `json:"exercise_id"`
	SortOrder  int                  `json:"sort_order,omitzero"`
	Notes      string               `json:"notes,omitzero"`
	Sets       []WorkoutSetResponse `json:"sets,omitzero"`
}

type WorkoutSetResponse struct {
	ID          uuid.UUID `json:"id"`
	SetNumber   int       `json:"set_number,omitzero"`
	WeightKg    float64   `json:"weight_kg"`
	Reps        int       `json:"reps"`
	RPE         *int      `json:"rpe,omitzero"`
	RestSeconds *int      `json:"rest_seconds,omitzero"`
	IsWarmup    bool      `json:"is_warmup,omitzero"`
}

type LogSetResponse struct {
	WorkoutSetResponse

	IsPR         bool    `json:"is_pr,omitzero"`
	Estimated1RM float64 `json:"estimated_1rm"`
}
