// Package exercise contains the exercise use cases and the interfaces the
// service consumes, declared where they are used per ISP.
package exercise

import (
	"context"

	"github.com/google/uuid"

	domainexercise "github.com/vladgrskkh/onerep-api/internal/domain/exercise"
)

type ExerciseRepository interface {
	List(ctx context.Context, filter domainexercise.ExerciseFilter) ([]domainexercise.Exercise, error)
	FindByID(ctx context.Context, id uuid.UUID) (domainexercise.Exercise, error)
	Create(ctx context.Context, ex domainexercise.Exercise) (domainexercise.Exercise, error)
	Update(ctx context.Context, ex domainexercise.Exercise) (domainexercise.Exercise, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

type MuscleGroupRepository interface {
	List(ctx context.Context) ([]domainexercise.MuscleGroup, error)
}
