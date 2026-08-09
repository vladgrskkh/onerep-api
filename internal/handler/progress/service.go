package progress

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainbodyweight "github.com/vladgrskkh/onerep-api/internal/domain/bodyweight"
	domainprogress "github.com/vladgrskkh/onerep-api/internal/domain/progress"
	servicebodyweight "github.com/vladgrskkh/onerep-api/internal/service/bodyweight"
)

// ProgressService is the progress read-model contract consumed by the
// handler.
type ProgressService interface {
	Get1RM(ctx context.Context, userID, exerciseID uuid.UUID, from, to *time.Time) ([]domainprogress.Progress1RM, error)
	GetVolume(ctx context.Context, userID uuid.UUID, from, to *time.Time) ([]domainprogress.ProgressVolume, error)
}

// BodyWeightService is the body weight contract consumed by the handler.
type BodyWeightService interface {
	LogBodyWeight(ctx context.Context, cmd servicebodyweight.LogBodyWeightCommand) (domainbodyweight.BodyWeight, error)
	ListBodyWeight(ctx context.Context, userID uuid.UUID, since *time.Time) ([]domainbodyweight.BodyWeight, error)
}
