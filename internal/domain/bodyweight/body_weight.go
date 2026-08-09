package bodyweight

import (
	"time"

	"github.com/google/uuid"
)

type BodyWeight struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	WeightKg   float64
	MeasuredAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Version    int
}

func NewBodyWeight(userID uuid.UUID, weightKg float64, measuredAt time.Time) BodyWeight {
	now := time.Now()
	return BodyWeight{
		ID:         uuid.Must(uuid.NewV7()),
		UserID:     userID,
		WeightKg:   weightKg,
		MeasuredAt: measuredAt,
		CreatedAt:  now,
		UpdatedAt:  now,
		Version:    1,
	}
}
