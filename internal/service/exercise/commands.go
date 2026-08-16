package exercise

import (
	"time"

	"github.com/google/uuid"

	domainexercise "github.com/vladgrskkh/onerep-api/internal/domain/exercise"
)

// CreateExerciseCommand carries the fields needed to create a user exercise.
type CreateExerciseCommand struct {
	Name           string
	Description    string
	Notes          string
	UserID         uuid.UUID
	MuscleGroupIDs []int
}

// UpdateExerciseCommand carries the fields to patch on an existing exercise.
// Pointers distinguish absent fields from empty ones.
type UpdateExerciseCommand struct {
	ID             uuid.UUID
	Name           *string
	Description    *string
	Notes          *string
	MuscleGroupIDs *[]int
}

// ListExercisesCommand carries the query filters for listing exercises. A
// zero Since means no time filter.
type ListExercisesCommand struct {
	Search      string
	MuscleGroup string
	Since       time.Time
}

// UploadExerciseMediaCommand carries the fields needed to register an
// uploaded media object for an exercise and presign its upload. The sort
// order and S3 key are computed by the service.
type UploadExerciseMediaCommand struct {
	ExerciseID  uuid.UUID
	MediaType   domainexercise.MediaType
	ContentType string
}

// Filter converts the command into the domain list filter.
func (c ListExercisesCommand) Filter() domainexercise.ExerciseFilter {
	return domainexercise.ExerciseFilter{
		Search:      c.Search,
		MuscleGroup: c.MuscleGroup,
		Since:       c.Since,
	}
}
