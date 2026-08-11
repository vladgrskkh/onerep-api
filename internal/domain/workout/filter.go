package workout

import (
	"time"

	"github.com/google/uuid"
)

type WorkoutFilter struct {
	UserID *uuid.UUID
	Since  time.Time
}
