package bodyweight

import (
	"time"

	"github.com/google/uuid"
)

// LogBodyWeightCommand carries the fields needed to log a body weight.
// MeasuredAt defaults to the current time when nil.
type LogBodyWeightCommand struct {
	UserID     uuid.UUID
	WeightKg   float64
	MeasuredAt *time.Time
}
