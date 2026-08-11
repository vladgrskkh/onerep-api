package template

import "github.com/google/uuid"

// TemplateExerciseCommand carries a single exercise within a template.
type TemplateExerciseCommand struct {
	ExerciseID  uuid.UUID
	PlannedSets int
}

// CreateTemplateCommand carries the fields needed to create a template.
type CreateTemplateCommand struct {
	Name        string
	Description string
	UserID      uuid.UUID
	Exercises   []TemplateExerciseCommand
}

// UpdateTemplateCommand carries the fields to patch on an existing template.
// Pointers distinguish absent fields from empty ones.
type UpdateTemplateCommand struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Name        *string
	Description *string
	Exercises   *[]TemplateExerciseCommand
}
