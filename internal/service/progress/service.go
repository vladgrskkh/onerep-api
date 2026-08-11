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

// ProgressService reads the user's materialized progress.
type ProgressService struct {
	progress ProgressRepository
}

func NewProgressService(progress ProgressRepository) *ProgressService {
	return &ProgressService{progress: progress}
}

// Get1RM returns the user's estimated one-rep max history for an exercise,
// optionally filtered by date range.
func (s *ProgressService) Get1RM(
	ctx context.Context,
	userID, exerciseID uuid.UUID,
	from, to *time.Time,
) ([]*domainprogress.Progress1RM, error) {
	if userID == uuid.Nil {
		return nil, domainprogress.ErrInvalidUserID
	}
	if exerciseID == uuid.Nil {
		return nil, domainprogress.ErrInvalidExerciseID
	}
	progress, err := s.progress.Get1RM(ctx, exerciseID, userID, from, to)
	if err != nil {
		return nil, err
	}
	out := make([]*domainprogress.Progress1RM, len(progress))
	for i := range progress {
		out[i] = &progress[i]
	}
	return out, nil
}

// GetVolume returns the user's total volume per muscle group, optionally
// filtered by date range.
func (s *ProgressService) GetVolume(
	ctx context.Context,
	userID uuid.UUID,
	from, to *time.Time,
) ([]*domainprogress.ProgressVolume, error) {
	if userID == uuid.Nil {
		return nil, domainprogress.ErrInvalidUserID
	}
	volumes, err := s.progress.GetVolume(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}
	out := make([]*domainprogress.ProgressVolume, len(volumes))
	for i := range volumes {
		out[i] = &volumes[i]
	}
	return out, nil
}
