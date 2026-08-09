package workout

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainworkout "github.com/vladgrskkh/onerep-api/internal/domain/workout"
	serviceworkout "github.com/vladgrskkh/onerep-api/internal/service/workout"
)

// WorkoutService is the workout use-case contract consumed by the handler.
type WorkoutService interface {
	Start(ctx context.Context, cmd serviceworkout.StartWorkoutCommand) (domainworkout.Workout, error)
	GetActive(ctx context.Context, userID uuid.UUID) (domainworkout.Workout, error)
	Get(ctx context.Context, id, userID uuid.UUID) (domainworkout.Workout, error)
	List(ctx context.Context, userID uuid.UUID, since *time.Time) ([]domainworkout.Workout, error)
	AddExercise(
		ctx context.Context,
		cmd serviceworkout.AddExerciseCommand,
		userID uuid.UUID,
	) (domainworkout.WorkoutExercise, error)
	LogSet(ctx context.Context, cmd serviceworkout.LogSetCommand, userID uuid.UUID) (serviceworkout.SetResult, error)
	Finish(
		ctx context.Context,
		cmd serviceworkout.FinishWorkoutCommand,
		userID uuid.UUID,
	) (domainworkout.Workout, error)
}
