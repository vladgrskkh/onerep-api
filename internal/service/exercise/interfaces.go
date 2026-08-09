package exercise

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainexercise "github.com/vladgrskkh/onerep-api/internal/domain/exercise"
)

//nolint:revive // mandated name; the package name matches the domain it serves
type ExerciseRepository interface {
	List(ctx context.Context, filter ExerciseFilter) ([]domainexercise.Exercise, error)
	FindByID(ctx context.Context, id uuid.UUID) (domainexercise.Exercise, error)
	Create(ctx context.Context, ex domainexercise.Exercise) (domainexercise.Exercise, error)
	Update(ctx context.Context, ex domainexercise.Exercise) (domainexercise.Exercise, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

//nolint:revive // mandated name; the package name matches the domain it serves
type ExerciseFilter struct {
	Search      string
	MuscleGroup string
	Since       *time.Time
	IsBuiltIn   *bool
}

type MuscleGroupRepository interface {
	List(ctx context.Context) ([]domainexercise.MuscleGroup, error)
}
