package bodyweight

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainbodyweight "github.com/vladgrskkh/onerep-api/internal/domain/bodyweight"
)

type BodyWeightRepository interface {
	List(ctx context.Context, userID uuid.UUID, since time.Time) ([]*domainbodyweight.BodyWeight, error)
	Create(ctx context.Context, bw domainbodyweight.BodyWeight) (*domainbodyweight.BodyWeight, error)
}

// BodyWeightService reads and writes the user's body weight entries.
type BodyWeightService struct {
	weights BodyWeightRepository
}

func NewBodyWeightService(weights BodyWeightRepository) *BodyWeightService {
	return &BodyWeightService{weights: weights}
}

// LogBodyWeight records a body weight entry for the user. The measurement
// time defaults to now when the command does not carry one.
func (s *BodyWeightService) LogBodyWeight(
	ctx context.Context,
	cmd LogBodyWeightCommand,
) (*domainbodyweight.BodyWeight, error) {
	measuredAt := time.Now()
	if !cmd.MeasuredAt.IsZero() {
		measuredAt = cmd.MeasuredAt
	}
	bw, err := domainbodyweight.NewBodyWeight(cmd.UserID, cmd.WeightKg, measuredAt)
	if err != nil {
		return nil, err
	}
	return s.weights.Create(ctx, bw)
}

// ListBodyWeight returns the user's body weight entries, optionally only
// those updated after since.
func (s *BodyWeightService) ListBodyWeight(
	ctx context.Context,
	userID uuid.UUID,
	since time.Time,
) ([]*domainbodyweight.BodyWeight, error) {
	if userID == uuid.Nil {
		return nil, domainbodyweight.ErrInvalidUserID
	}
	return s.weights.List(ctx, userID, since)
}
