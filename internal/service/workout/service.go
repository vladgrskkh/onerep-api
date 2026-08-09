package workout

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainworkout "github.com/vladgrskkh/onerep-api/internal/domain/workout"
)

type WorkoutRepository interface {
	List(ctx context.Context, filter domainworkout.WorkoutFilter) ([]domainworkout.Workout, error)
	FindByID(ctx context.Context, id uuid.UUID) (domainworkout.Workout, error)
	Create(ctx context.Context, w domainworkout.Workout) (domainworkout.Workout, error)
	AddExercise(ctx context.Context, workoutID, exerciseID uuid.UUID) (domainworkout.WorkoutExercise, error)
	LogSet(
		ctx context.Context,
		workoutExerciseID uuid.UUID,
		set domainworkout.WorkoutSet,
	) (domainworkout.WorkoutSet, error)
	Update(ctx context.Context, w domainworkout.Workout) (domainworkout.Workout, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (domainworkout.Workout, error)
}
