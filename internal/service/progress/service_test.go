package progress_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	domainprogress "github.com/vladgrskkh/onerep-api/internal/domain/progress"
	serviceprogress "github.com/vladgrskkh/onerep-api/internal/service/progress"
	progressmocks "github.com/vladgrskkh/onerep-api/internal/service/progress/mocks"
)

type ServiceTestSuite struct {
	suite.Suite

	svc  *serviceprogress.ProgressService
	repo *progressmocks.MockProgressRepository
}

func (s *ServiceTestSuite) SetupTest() {
	s.repo = progressmocks.NewMockProgressRepository(s.T())
	s.svc = serviceprogress.NewProgressService(s.repo)
}

func (s *ServiceTestSuite) TestGet1RM_Success() {
	userID := uuid.Must(uuid.NewV7())
	exerciseID := uuid.Must(uuid.NewV7())
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	expected := []*domainprogress.Progress1RM{
		{ExerciseID: exerciseID, UserID: userID, Date: from, Estimated1RM: 120},
	}
	s.repo.EXPECT().Get1RM(mock.Anything, exerciseID, userID, from, to).Return(expected, nil)

	got, err := s.svc.Get1RM(context.Background(), userID, exerciseID, from, to)
	s.Require().NoError(err)
	s.Require().Len(got, 1)
	s.Equal(expected[0], got[0])
}

func (s *ServiceTestSuite) TestGet1RM_NilUserID() {
	_, err := s.svc.Get1RM(
		context.Background(),
		uuid.Nil,
		uuid.Must(uuid.NewV7()),
		time.Time{},
		time.Time{},
	)
	s.Require().ErrorIs(err, domainprogress.ErrInvalidUserID)
	s.repo.AssertNotCalled(s.T(), "Get1RM", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestGet1RM_NilExerciseID() {
	_, err := s.svc.Get1RM(
		context.Background(),
		uuid.Must(uuid.NewV7()),
		uuid.Nil,
		time.Time{},
		time.Time{},
	)
	s.Require().ErrorIs(err, domainprogress.ErrInvalidExerciseID)
	s.repo.AssertNotCalled(s.T(), "Get1RM", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestGetVolume_Success() {
	userID := uuid.Must(uuid.NewV7())
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	expected := []*domainprogress.ProgressVolume{
		{MuscleGroupID: 1, UserID: userID, Date: from, TotalKG: 5000},
	}
	s.repo.EXPECT().GetVolume(mock.Anything, userID, from, time.Time{}).Return(expected, nil)

	got, err := s.svc.GetVolume(context.Background(), userID, from, time.Time{})
	s.Require().NoError(err)
	s.Require().Len(got, 1)
	s.Equal(expected[0], got[0])
}

func (s *ServiceTestSuite) TestGetVolume_NilUserID() {
	_, err := s.svc.GetVolume(context.Background(), uuid.Nil, time.Time{}, time.Time{})
	s.Require().ErrorIs(err, domainprogress.ErrInvalidUserID)
	s.repo.AssertNotCalled(s.T(), "GetVolume", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestServiceSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(ServiceTestSuite))
}
