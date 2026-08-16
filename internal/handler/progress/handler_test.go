package progress_test

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	domainbodyweight "github.com/vladgrskkh/onerep-api/internal/domain/bodyweight"
	domainprogress "github.com/vladgrskkh/onerep-api/internal/domain/progress"
	progresshandler "github.com/vladgrskkh/onerep-api/internal/handler/progress"
	"github.com/vladgrskkh/onerep-api/internal/handler/progress/dto"
	progressmocks "github.com/vladgrskkh/onerep-api/internal/handler/progress/mocks"
	"github.com/vladgrskkh/onerep-api/internal/handler/testutil"
	servicebodyweight "github.com/vladgrskkh/onerep-api/internal/service/bodyweight"
)

// errRepo simulates an unexpected repository failure.
var errRepo = errors.New("repository failure")

type HandlerTestSuite struct {
	suite.Suite

	handler       *progresshandler.ProgressHandler
	progressSvc   *progressmocks.MockProgressService
	bodyWeightSvc *progressmocks.MockBodyWeightService
	router        *chi.Mux
	userID        uuid.UUID
}

func (s *HandlerTestSuite) SetupTest() {
	s.userID = uuid.Must(uuid.NewV7())
	s.progressSvc = progressmocks.NewMockProgressService(s.T())
	s.bodyWeightSvc = progressmocks.NewMockBodyWeightService(s.T())
	s.handler = progresshandler.NewProgressHandler(
		s.progressSvc,
		s.bodyWeightSvc,
		slog.New(slog.DiscardHandler),
	)

	s.router = chi.NewRouter()
	s.router.Get("/v1/progress/1rm", s.handler.Get1RM)
	s.router.Get("/v1/progress/volume", s.handler.GetVolume)
	s.router.Get("/v1/progress/body-weight", s.handler.GetBodyWeight)
	s.router.Post("/v1/progress/body-weight", s.handler.LogBodyWeight)
}

func (s *HandlerTestSuite) get1RM(query string, userID uuid.UUID) *httptest.ResponseRecorder {
	return testutil.Serve(
		s.router,
		http.MethodGet,
		testutil.URL(s.router, "/v1/progress/1rm")+query,
		"",
		userID,
	)
}

func (s *HandlerTestSuite) getVolume(query string, userID uuid.UUID) *httptest.ResponseRecorder {
	return testutil.Serve(
		s.router,
		http.MethodGet,
		testutil.URL(s.router, "/v1/progress/volume")+query,
		"",
		userID,
	)
}

func (s *HandlerTestSuite) getBodyWeight(
	query string,
	userID uuid.UUID,
) *httptest.ResponseRecorder {
	return testutil.Serve(
		s.router,
		http.MethodGet,
		testutil.URL(s.router, "/v1/progress/body-weight")+query,
		"",
		userID,
	)
}

func (s *HandlerTestSuite) logBodyWeight(body string, userID uuid.UUID) *httptest.ResponseRecorder {
	return testutil.Serve(
		s.router,
		http.MethodPost,
		testutil.URL(s.router, "/v1/progress/body-weight"),
		body,
		userID,
	)
}

func (s *HandlerTestSuite) TestGet1RM_Success() {
	exerciseID := uuid.Must(uuid.NewV7())
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	s.progressSvc.EXPECT().Get1RM(mock.Anything, s.userID, exerciseID, from, to).
		Return([]*domainprogress.Progress1RM{
			{ExerciseID: exerciseID, UserID: s.userID, Date: from, Estimated1RM: 120},
		}, nil)

	w := s.get1RM(
		"?exercise_id="+exerciseID.String()+
			"&from=2026-07-01T00%3A00%3A00Z&to=2026-08-01T00%3A00%3A00Z",
		s.userID,
	)

	s.Equal(http.StatusOK, w.Code)
	var resp []dto.OneRMResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Require().Len(resp, 1)
	s.True(resp[0].Date.Equal(from))
	s.InDelta(120.0, resp[0].Estimated1RM, 0.0001)
}

func (s *HandlerTestSuite) TestGet1RM_NoDateRange() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.progressSvc.EXPECT().
		Get1RM(mock.Anything, s.userID, exerciseID, time.Time{}, time.Time{}).
		Return([]*domainprogress.Progress1RM{}, nil)

	w := s.get1RM("?exercise_id="+exerciseID.String(), s.userID)

	s.Equal(http.StatusOK, w.Code)
}

func (s *HandlerTestSuite) TestGet1RM_InvalidExerciseID() {
	w := s.get1RM("?exercise_id=not-a-uuid", s.userID)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INVALID_EXERCISE_ID", string(resp.Error.Code))
	s.progressSvc.AssertNotCalled(
		s.T(),
		"Get1RM",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}

func (s *HandlerTestSuite) TestGet1RM_InvalidFrom() {
	exerciseID := uuid.Must(uuid.NewV7())
	w := s.get1RM("?exercise_id="+exerciseID.String()+"&from=not-a-timestamp", s.userID)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INVALID_DATE_RANGE", string(resp.Error.Code))
	s.progressSvc.AssertNotCalled(
		s.T(),
		"Get1RM",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}

func (s *HandlerTestSuite) TestGet1RM_InvalidTo() {
	exerciseID := uuid.Must(uuid.NewV7())
	w := s.get1RM("?exercise_id="+exerciseID.String()+"&to=not-a-timestamp", s.userID)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INVALID_DATE_RANGE", string(resp.Error.Code))
	s.progressSvc.AssertNotCalled(
		s.T(),
		"Get1RM",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}

func (s *HandlerTestSuite) TestGet1RM_InvertedRange() {
	exerciseID := uuid.Must(uuid.NewV7())
	w := s.get1RM(
		"?exercise_id="+exerciseID.String()+
			"&from=2026-01-01T00%3A00%3A00Z&to=2025-01-01T00%3A00%3A00Z",
		s.userID,
	)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INVALID_DATE_RANGE", string(resp.Error.Code))
	s.progressSvc.AssertNotCalled(
		s.T(),
		"Get1RM",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}

func (s *HandlerTestSuite) TestGet1RM_ServiceError() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.progressSvc.EXPECT().
		Get1RM(mock.Anything, s.userID, exerciseID, time.Time{}, time.Time{}).
		Return(nil, domainprogress.ErrInvalidExerciseID)

	w := s.get1RM("?exercise_id="+exerciseID.String(), s.userID)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INVALID_EXERCISE_ID", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestGet1RM_InternalError() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.progressSvc.EXPECT().
		Get1RM(mock.Anything, s.userID, exerciseID, time.Time{}, time.Time{}).
		Return(nil, errRepo)

	w := s.get1RM("?exercise_id="+exerciseID.String(), s.userID)

	s.Equal(http.StatusInternalServerError, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INTERNAL_ERROR", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestGetVolume_Success() {
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	s.progressSvc.EXPECT().GetVolume(mock.Anything, s.userID, from, time.Time{}).
		Return([]*domainprogress.ProgressVolume{
			{MuscleGroupID: 1, UserID: s.userID, Date: from, TotalKG: 5000},
		}, nil)

	w := s.getVolume("?from=2026-07-01T00%3A00%3A00Z", s.userID)

	s.Equal(http.StatusOK, w.Code)
	var resp []dto.VolumeResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Require().Len(resp, 1)
	s.True(resp[0].Date.Equal(from))
	s.Equal(1, resp[0].MuscleGroupID)
	s.InDelta(5000.0, resp[0].TotalKG, 0.0001)
}

func (s *HandlerTestSuite) TestGetVolume_InvalidTo() {
	w := s.getVolume("?to=not-a-timestamp", s.userID)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INVALID_DATE_RANGE", string(resp.Error.Code))
	s.progressSvc.AssertNotCalled(
		s.T(),
		"GetVolume",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}

func (s *HandlerTestSuite) TestGetVolume_InvertedRange() {
	w := s.getVolume(
		"?from=2026-01-01T00%3A00%3A00Z&to=2025-01-01T00%3A00%3A00Z",
		s.userID,
	)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INVALID_DATE_RANGE", string(resp.Error.Code))
	s.progressSvc.AssertNotCalled(
		s.T(),
		"GetVolume",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}

func (s *HandlerTestSuite) TestGetBodyWeight_Success() {
	since := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	bwID := uuid.Must(uuid.NewV7())
	s.bodyWeightSvc.EXPECT().ListBodyWeight(mock.Anything, s.userID, since).
		Return([]*domainbodyweight.BodyWeight{
			{ID: bwID, UserID: s.userID, WeightKg: 80, MeasuredAt: since, CreatedAt: since},
		}, nil)

	w := s.getBodyWeight("?since=2026-08-01T00%3A00%3A00Z", s.userID)

	s.Equal(http.StatusOK, w.Code)
	var resp []dto.BodyWeightResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Require().Len(resp, 1)
	s.Equal(bwID, resp[0].ID)
	s.InDelta(80.0, resp[0].WeightKg, 0.0001)
	s.True(resp[0].MeasuredAt.Equal(since))
}

func (s *HandlerTestSuite) TestGetBodyWeight_InvalidSince() {
	w := s.getBodyWeight("?since=not-a-timestamp", s.userID)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INVALID_SINCE", string(resp.Error.Code))
	s.bodyWeightSvc.AssertNotCalled(
		s.T(),
		"ListBodyWeight",
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}

func (s *HandlerTestSuite) TestLogBodyWeight_Success() {
	measuredAt := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	bwID := uuid.Must(uuid.NewV7())
	created := &domainbodyweight.BodyWeight{
		ID:         bwID,
		UserID:     s.userID,
		WeightKg:   80,
		MeasuredAt: measuredAt,
		CreatedAt:  measuredAt,
	}
	s.bodyWeightSvc.EXPECT().LogBodyWeight(mock.Anything, servicebodyweight.LogBodyWeightCommand{
		UserID:     s.userID,
		WeightKg:   80,
		MeasuredAt: measuredAt,
	}).Return(created, nil)

	w := s.logBodyWeight(`{"weight_kg":80,"measured_at":"2026-08-01T10:00:00Z"}`, s.userID)

	s.Equal(http.StatusCreated, w.Code)
	var resp dto.BodyWeightResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(bwID, resp.ID)
	s.InDelta(80.0, resp.WeightKg, 0.0001)
	s.True(resp.MeasuredAt.Equal(measuredAt))
}

func (s *HandlerTestSuite) TestLogBodyWeight_NoMeasuredAt() {
	s.bodyWeightSvc.EXPECT().
		LogBodyWeight(mock.Anything, mock.MatchedBy(func(cmd servicebodyweight.LogBodyWeightCommand) bool {
			return cmd.UserID == s.userID && cmd.WeightKg == 80 && cmd.MeasuredAt.IsZero()
		})).
		Return(&domainbodyweight.BodyWeight{ID: uuid.Must(uuid.NewV7())}, nil)

	w := s.logBodyWeight(`{"weight_kg":80}`, s.userID)

	s.Equal(http.StatusCreated, w.Code)
}

func (s *HandlerTestSuite) TestLogBodyWeight_ValidationError() {
	w := s.logBodyWeight(`{}`, s.userID)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("VALIDATION_ERROR", string(resp.Error.Code))
	s.Require().Len(resp.Error.Details, 1)
	s.Equal("weight_kg", resp.Error.Details[0].Field)
	s.bodyWeightSvc.AssertNotCalled(s.T(), "LogBodyWeight", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestLogBodyWeight_InvalidJSON() {
	w := s.logBodyWeight(`{"weight_kg":`, s.userID)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INVALID_REQUEST_BODY", string(resp.Error.Code))
	s.bodyWeightSvc.AssertNotCalled(s.T(), "LogBodyWeight", mock.Anything, mock.Anything)
}

func TestHandlerSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(HandlerTestSuite))
}
