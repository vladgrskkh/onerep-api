package bodyweight_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	domainbodyweight "github.com/vladgrskkh/onerep-api/internal/domain/bodyweight"
	servicebodyweight "github.com/vladgrskkh/onerep-api/internal/service/bodyweight"
	bodyweightmocks "github.com/vladgrskkh/onerep-api/internal/service/bodyweight/mocks"
)

type ServiceTestSuite struct {
	suite.Suite

	svc  *servicebodyweight.BodyWeightService
	repo *bodyweightmocks.MockBodyWeightRepository
}

func (s *ServiceTestSuite) SetupTest() {
	s.repo = bodyweightmocks.NewMockBodyWeightRepository(s.T())
	s.svc = servicebodyweight.NewBodyWeightService(s.repo)
}

func (s *ServiceTestSuite) TestLogBodyWeight_Success() {
	userID := uuid.Must(uuid.NewV7())
	measuredAt := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	s.repo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(bw domainbodyweight.BodyWeight) bool {
			return bw.UserID == userID &&
				bw.WeightKg == 80 &&
				bw.MeasuredAt.Equal(measuredAt)
		})).
		RunAndReturn(func(_ context.Context, bw domainbodyweight.BodyWeight) (*domainbodyweight.BodyWeight, error) {
			return &bw, nil
		})

	created, err := s.svc.LogBodyWeight(context.Background(), servicebodyweight.LogBodyWeightCommand{
		UserID:     userID,
		WeightKg:   80,
		MeasuredAt: measuredAt,
	})
	s.Require().NoError(err)
	s.Equal(userID, created.UserID)
	s.InDelta(80.0, created.WeightKg, 0.0001)
	s.True(created.MeasuredAt.Equal(measuredAt))
	s.Equal(1, created.Version)
}

func (s *ServiceTestSuite) TestLogBodyWeight_DefaultsMeasuredAt() {
	userID := uuid.Must(uuid.NewV7())
	var measuredAt time.Time
	s.repo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(bw domainbodyweight.BodyWeight) bool {
			measuredAt = bw.MeasuredAt
			return bw.UserID == userID && bw.WeightKg == 80
		})).
		Return(&domainbodyweight.BodyWeight{}, nil)

	_, err := s.svc.LogBodyWeight(context.Background(), servicebodyweight.LogBodyWeightCommand{
		UserID:   userID,
		WeightKg: 80,
	})
	s.Require().NoError(err)
	s.WithinDuration(time.Now(), measuredAt, time.Second)
}

func (s *ServiceTestSuite) TestLogBodyWeight_InvalidWeight() {
	userID := uuid.Must(uuid.NewV7())
	_, err := s.svc.LogBodyWeight(context.Background(), servicebodyweight.LogBodyWeightCommand{
		UserID:   userID,
		WeightKg: 0,
	})
	s.Require().ErrorIs(err, domainbodyweight.ErrInvalidWeight)
	s.repo.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestLogBodyWeight_NilUserID() {
	_, err := s.svc.LogBodyWeight(context.Background(), servicebodyweight.LogBodyWeightCommand{
		WeightKg: 80,
	})
	s.Require().ErrorIs(err, domainbodyweight.ErrInvalidUserID)
	s.repo.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestListBodyWeight_Success() {
	userID := uuid.Must(uuid.NewV7())
	since := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	expected := []*domainbodyweight.BodyWeight{{UserID: userID, WeightKg: 80, MeasuredAt: since}}
	s.repo.EXPECT().List(mock.Anything, userID, since).Return(expected, nil)

	got, err := s.svc.ListBodyWeight(context.Background(), userID, since)
	s.Require().NoError(err)
	s.Require().Len(got, 1)
	s.Equal(expected[0], got[0])
}

func (s *ServiceTestSuite) TestListBodyWeight_NilUserID() {
	_, err := s.svc.ListBodyWeight(context.Background(), uuid.Nil, time.Time{})
	s.Require().ErrorIs(err, domainbodyweight.ErrInvalidUserID)
	s.repo.AssertNotCalled(s.T(), "List", mock.Anything, mock.Anything, mock.Anything)
}

func TestServiceSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(ServiceTestSuite))
}
