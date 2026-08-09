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

	p, err := progress.NewProgress1RM(exerciseID, userID, date, 120)

	s.Require().NoError(err)
	s.Equal(exerciseID, p.ExerciseID)
	s.Equal(userID, p.UserID)
	s.Equal(date, p.Date)
	s.InEpsilon(120.0, p.Estimated1RM, 1e-6)
}

func (s *ProgressTestSuite) TestNewProgress1RM_Validation() {
	exerciseID := uuid.MustParse("9f3b8f3e-4f1d-4f6a-8b3e-3a2f5c9d1e2a")
	userID := uuid.MustParse("1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d")
	date := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		exerciseID   uuid.UUID
		userID       uuid.UUID
		estimated1RM float64
		wantErr      error
	}{
		{name: "valid", exerciseID: exerciseID, userID: userID, estimated1RM: 120},
		{
			name:         "nil exercise id",
			exerciseID:   uuid.Nil,
			userID:       userID,
			estimated1RM: 120,
			wantErr:      progress.ErrInvalidExerciseID,
		},
		{
			name:         "nil user id",
			exerciseID:   exerciseID,
			userID:       uuid.Nil,
			estimated1RM: 120,
			wantErr:      progress.ErrInvalidUserID,
		},
		{
			name:         "zero estimated 1rm",
			exerciseID:   exerciseID,
			userID:       userID,
			estimated1RM: 0,
			wantErr:      progress.ErrInvalidEstimated1RM,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			p, err := progress.NewProgress1RM(tt.exerciseID, tt.userID, date, tt.estimated1RM)
			if tt.wantErr != nil {
				s.Require().ErrorIs(err, tt.wantErr)
				s.Equal(uuid.Nil, p.ExerciseID)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.exerciseID, p.ExerciseID)
			s.Equal(tt.userID, p.UserID)
		})
	}
}

func (s *ProgressTestSuite) TestNewProgressVolume() {
	userID := uuid.MustParse("1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d")
	date := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)

	p, err := progress.NewProgressVolume(1, userID, date, 5000)

	s.Require().NoError(err)
	s.Equal(1, p.MuscleGroupID)
	s.Equal(userID, p.UserID)
	s.Equal(date, p.Date)
	s.InEpsilon(5000.0, p.TotalKG, 1e-6)
}

func (s *ProgressTestSuite) TestNewProgressVolume_Validation() {
	userID := uuid.MustParse("1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d")
	date := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		muscleGroupID int
		userID        uuid.UUID
		totalKG       float64
		wantErr       error
	}{
		{name: "valid", muscleGroupID: 1, userID: userID, totalKG: 5000},
		{
			name:          "nil user id",
			muscleGroupID: 1,
			userID:        uuid.Nil,
			totalKG:       5000,
			wantErr:       progress.ErrInvalidUserID,
		},
		{
			name:          "zero muscle group id",
			muscleGroupID: 0,
			userID:        userID,
			totalKG:       5000,
			wantErr:       progress.ErrInvalidMuscleGroupID,
		},
		{
			name:          "zero total kg",
			muscleGroupID: 1,
			userID:        userID,
			totalKG:       0,
			wantErr:       progress.ErrInvalidTotalKG,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			p, err := progress.NewProgressVolume(tt.muscleGroupID, tt.userID, date, tt.totalKG)
			if tt.wantErr != nil {
				s.Require().ErrorIs(err, tt.wantErr)
				s.Equal(0, p.MuscleGroupID)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.muscleGroupID, p.MuscleGroupID)
			s.Equal(tt.userID, p.UserID)
		})
	}
}

func TestProgressSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(ProgressTestSuite))
}
