package bodyweight_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/vladgrskkh/onerep-api/internal/domain/bodyweight"
)

type BodyWeightTestSuite struct {
	suite.Suite
}

func (s *BodyWeightTestSuite) TestNewBodyWeight() {
	userID := uuid.MustParse("9f3b8f3e-4f1d-4f6a-8b3e-3a2f5c9d1e2a")
	measuredAt := time.Date(2026, 8, 9, 8, 0, 0, 0, time.UTC)

	bw := bodyweight.NewBodyWeight(userID, 78.5, measuredAt)

	s.NotEqual(uuid.Nil, bw.ID)
	s.Equal(uuid.Version(7), bw.ID.Version())
	s.Equal(userID, bw.UserID)
	s.InEpsilon(78.5, bw.WeightKg, 1e-6)
	s.Equal(measuredAt, bw.MeasuredAt)
	s.False(bw.CreatedAt.IsZero())
	s.False(bw.UpdatedAt.IsZero())
	s.WithinDuration(time.Now(), bw.CreatedAt, time.Minute)
	s.Equal(1, bw.Version)
}

func TestBodyWeightSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(BodyWeightTestSuite))
}
