package exercise

import "time"

type ExerciseFilter struct {
	Search      string
	MuscleGroup string
	Since       time.Time
	IsBuiltIn   *bool
}
