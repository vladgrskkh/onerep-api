package bodyweight

import (
	"time"

	"github.com/google/uuid"
)

// LogBodyWeightCommand carries the fields needed to log a body weight.
// A zero MeasuredAt defaults to the current time.
type LogBodyWeightCommand struct {
	UserID     uuid.UUID
	WeightKg   float64
	MeasuredAt time.Time
}
