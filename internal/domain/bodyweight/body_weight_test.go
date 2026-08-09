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

	bw, err := bodyweight.NewBodyWeight(userID, 78.5, measuredAt)

	s.Require().NoError(err)
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

func (s *BodyWeightTestSuite) TestNewBodyWeight_Validation() {
	userID := uuid.MustParse("9f3b8f3e-4f1d-4f6a-8b3e-3a2f5c9d1e2a")
	measuredAt := time.Date(2026, 8, 9, 8, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		userID   uuid.UUID
		weightKg float64
		wantErr  error
	}{
		{name: "valid", userID: userID, weightKg: 78.5},
		{name: "nil user id", userID: uuid.Nil, weightKg: 78.5, wantErr: bodyweight.ErrInvalidUserID},
		{name: "zero weight", userID: userID, weightKg: 0, wantErr: bodyweight.ErrInvalidWeight},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			bw, err := bodyweight.NewBodyWeight(tt.userID, tt.weightKg, measuredAt)
			if tt.wantErr != nil {
				s.Require().ErrorIs(err, tt.wantErr)
				s.Equal(uuid.Nil, bw.ID)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.userID, bw.UserID)
			s.InEpsilon(tt.weightKg, bw.WeightKg, 1e-6)
		})
	}
}

func TestBodyWeightSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(BodyWeightTestSuite))
}
