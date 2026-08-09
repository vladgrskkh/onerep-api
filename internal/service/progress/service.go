package progress

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainprogress "github.com/vladgrskkh/onerep-api/internal/domain/progress"
)

type ProgressRepository interface {
	Get1RM(ctx context.Context, exerciseID, userID uuid.UUID, from, to *time.Time) ([]domainprogress.Progress1RM, error)
	GetVolume(ctx context.Context, userID uuid.UUID, from, to *time.Time) ([]domainprogress.ProgressVolume, error)
	GetBest1RM(ctx context.Context, exerciseID, userID uuid.UUID) (float64, error)
	Upsert1RM(ctx context.Context, p domainprogress.Progress1RM) error
}
