package progress_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/vladgrskkh/onerep-api/internal/domain/progress"
)

type ProgressTestSuite struct {
	suite.Suite
}

func (s *ProgressTestSuite) TestNewProgress1RM() {
	exerciseID := uuid.MustParse("9f3b8f3e-4f1d-4f6a-8b3e-3a2f5c9d1e2a")
	userID := uuid.MustParse("1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d")
	date := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)

	p := progress.NewProgress1RM(exerciseID, userID, date, 120)

	s.Equal(exerciseID, p.ExerciseID)
	s.Equal(userID, p.UserID)
	s.Equal(date, p.Date)
	s.InEpsilon(120.0, p.Estimated1RM, 1e-6)
}

func (s *ProgressTestSuite) TestNewProgressVolume() {
	userID := uuid.MustParse("1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d")
	date := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)

	p := progress.NewProgressVolume(1, userID, date, 5000)

	s.Equal(1, p.MuscleGroupID)
	s.Equal(userID, p.UserID)
	s.Equal(date, p.Date)
	s.InEpsilon(5000.0, p.TotalKG, 1e-6)
}

func TestProgressSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(ProgressTestSuite))
}
