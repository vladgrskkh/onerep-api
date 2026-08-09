// Package bodyweight contains the body weight use cases and the interfaces
// the service consumes, declared where they are used per ISP.
package bodyweight

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainbodyweight "github.com/vladgrskkh/onerep-api/internal/domain/bodyweight"
)

type BodyWeightRepository interface {
	List(ctx context.Context, userID uuid.UUID, since *time.Time) ([]domainbodyweight.BodyWeight, error)
	Create(ctx context.Context, bw domainbodyweight.BodyWeight) (domainbodyweight.BodyWeight, error)
}
