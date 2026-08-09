package progress_test

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	domainbodyweight "github.com/vladgrskkh/onerep-api/internal/domain/bodyweight"
	domainprogress "github.com/vladgrskkh/onerep-api/internal/domain/progress"
	"github.com/vladgrskkh/onerep-api/internal/handler"
	progresshandler "github.com/vladgrskkh/onerep-api/internal/handler/progress"
	"github.com/vladgrskkh/onerep-api/internal/handler/progress/dto"
	progressmocks "github.com/vladgrskkh/onerep-api/internal/handler/progress/mocks"
	servicebodyweight "github.com/vladgrskkh/onerep-api/internal/service/bodyweight"
)

// errRepo simulates an unexpected repository failure.
var errRepo = errors.New("repository failure")

type HandlerTestSuite struct {
	suite.Suite

	handler       *progresshandler.ProgressHandler
	progressSvc   *progressmocks.MockProgressService
	bodyWeightSvc *progressmocks.MockBodyWeightService
}

func (s *HandlerTestSuite) SetupTest() {
	s.progressSvc = progressmocks.NewMockProgressService(s.T())
	s.bodyWeightSvc = progressmocks.NewMockBodyWeightService(s.T())
	logger := slog.New(slog.DiscardHandler)
	s.handler = progresshandler.NewProgressHandler(s.progressSvc, s.bodyWeightSvc, logger)
}

func (s *HandlerTestSuite) serve(method, path, body string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/progress/1rm", s.handler.Get1RM)
	mux.HandleFunc("GET /v1/progress/volume", s.handler.GetVolume)
	mux.HandleFunc("GET /v1/progress/body-weight", s.handler.GetBodyWeight)
	mux.HandleFunc("POST /v1/progress/body-weight", s.handler.LogBodyWeight)

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func (s *HandlerTestSuite) decodeError(w *httptest.ResponseRecorder) handler.ErrorResponse {
	var resp handler.ErrorResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	return resp
}

func (s *HandlerTestSuite) TestGet1RM_Success() {
	userID := uuid.Must(uuid.NewV7())
	exerciseID := uuid.Must(uuid.NewV7())
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	s.progressSvc.EXPECT().Get1RM(mock.Anything, userID, exerciseID, &from, &to).
		Return([]domainprogress.Progress1RM{
			{ExerciseID: exerciseID, UserID: userID, Date: from, Estimated1RM: 120},
		}, nil)

	req := httptest.NewRequest(
		http.MethodGet,
		"/v1/progress/1rm?exercise_id="+exerciseID.String()+
			"&from=2026-07-01T00%3A00%3A00Z&to=2026-08-01T00%3A00%3A00Z",
		nil,
	)
	req = req.WithContext(handler.WithUserID(req.Context(), userID))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/progress/1rm", s.handler.Get1RM)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
	var resp []dto.OneRMResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Require().Len(resp, 1)
	s.True(resp[0].Date.Equal(from))
	s.InDelta(120.0, resp[0].Estimated1RM, 0.0001)
}

func (s *HandlerTestSuite) TestGet1RM_NoDateRange() {
	userID := uuid.Must(uuid.NewV7())
	exerciseID := uuid.Must(uuid.NewV7())
	s.progressSvc.EXPECT().
		Get1RM(mock.Anything, userID, exerciseID, (*time.Time)(nil), (*time.Time)(nil)).
		Return([]domainprogress.Progress1RM{}, nil)

	req := httptest.NewRequest(
		http.MethodGet,
		"/v1/progress/1rm?exercise_id="+exerciseID.String(),
		nil,
	)
	req = req.WithContext(handler.WithUserID(req.Context(), userID))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/progress/1rm", s.handler.Get1RM)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
}

func (s *HandlerTestSuite) TestGet1RM_InvalidExerciseID() {
	w := s.serve(http.MethodGet, "/v1/progress/1rm?exercise_id=not-a-uuid", "")

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
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
	w := s.serve(
		http.MethodGet,
		"/v1/progress/1rm?exercise_id="+exerciseID.String()+"&from=not-a-timestamp",
		"",
	)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
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
	w := s.serve(
		http.MethodGet,
		"/v1/progress/1rm?exercise_id="+exerciseID.String()+"&to=not-a-timestamp",
		"",
	)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
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
	userID := uuid.Must(uuid.NewV7())
	exerciseID := uuid.Must(uuid.NewV7())
	s.progressSvc.EXPECT().
		Get1RM(mock.Anything, userID, exerciseID, (*time.Time)(nil), (*time.Time)(nil)).
		Return(nil, domainprogress.ErrInvalidExerciseID)

	req := httptest.NewRequest(
		http.MethodGet,
		"/v1/progress/1rm?exercise_id="+exerciseID.String(),
		nil,
	)
	req = req.WithContext(handler.WithUserID(req.Context(), userID))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/progress/1rm", s.handler.Get1RM)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("INVALID_EXERCISE_ID", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestGet1RM_InternalError() {
	userID := uuid.Must(uuid.NewV7())
	exerciseID := uuid.Must(uuid.NewV7())
	s.progressSvc.EXPECT().
		Get1RM(mock.Anything, userID, exerciseID, (*time.Time)(nil), (*time.Time)(nil)).
		Return(nil, errRepo)

	req := httptest.NewRequest(
		http.MethodGet,
		"/v1/progress/1rm?exercise_id="+exerciseID.String(),
		nil,
	)
	req = req.WithContext(handler.WithUserID(req.Context(), userID))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/progress/1rm", s.handler.Get1RM)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	s.Equal(http.StatusInternalServerError, w.Code)
	resp := s.decodeError(w)
	s.Equal("INTERNAL_ERROR", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestGetVolume_Success() {
	userID := uuid.Must(uuid.NewV7())
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	s.progressSvc.EXPECT().GetVolume(mock.Anything, userID, &from, (*time.Time)(nil)).
		Return([]domainprogress.ProgressVolume{
			{MuscleGroupID: 1, UserID: userID, Date: from, TotalKG: 5000},
		}, nil)

	req := httptest.NewRequest(
		http.MethodGet,
		"/v1/progress/volume?from=2026-07-01T00%3A00%3A00Z",
		nil,
	)
	req = req.WithContext(handler.WithUserID(req.Context(), userID))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/progress/volume", s.handler.GetVolume)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
	var resp []dto.VolumeResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Require().Len(resp, 1)
	s.True(resp[0].Date.Equal(from))
	s.Equal(1, resp[0].MuscleGroupID)
	s.InDelta(5000.0, resp[0].TotalKG, 0.0001)
}

func (s *HandlerTestSuite) TestGetVolume_InvalidTo() {
	w := s.serve(http.MethodGet, "/v1/progress/volume?to=not-a-timestamp", "")

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
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
	userID := uuid.Must(uuid.NewV7())
	since := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	bwID := uuid.Must(uuid.NewV7())
	s.bodyWeightSvc.EXPECT().ListBodyWeight(mock.Anything, userID, &since).
		Return([]domainbodyweight.BodyWeight{
			{ID: bwID, UserID: userID, WeightKg: 80, MeasuredAt: since, CreatedAt: since},
		}, nil)

	req := httptest.NewRequest(
		http.MethodGet,
		"/v1/progress/body-weight?since=2026-08-01T00%3A00%3A00Z",
		nil,
	)
	req = req.WithContext(handler.WithUserID(req.Context(), userID))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/progress/body-weight", s.handler.GetBodyWeight)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
	var resp []dto.BodyWeightResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Require().Len(resp, 1)
	s.Equal(bwID, resp[0].ID)
	s.InDelta(80.0, resp[0].WeightKg, 0.0001)
	s.True(resp[0].MeasuredAt.Equal(since))
}

func (s *HandlerTestSuite) TestGetBodyWeight_InvalidSince() {
	w := s.serve(http.MethodGet, "/v1/progress/body-weight?since=not-a-timestamp", "")

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
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
	userID := uuid.Must(uuid.NewV7())
	measuredAt := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	bwID := uuid.Must(uuid.NewV7())
	created := domainbodyweight.BodyWeight{
		ID:         bwID,
		UserID:     userID,
		WeightKg:   80,
		MeasuredAt: measuredAt,
		CreatedAt:  measuredAt,
	}
	s.bodyWeightSvc.EXPECT().LogBodyWeight(mock.Anything, servicebodyweight.LogBodyWeightCommand{
		UserID:     userID,
		WeightKg:   80,
		MeasuredAt: &measuredAt,
	}).Return(created, nil)

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/progress/body-weight",
		strings.NewReader(`{"weight_kg":80,"measured_at":"2026-08-01T10:00:00Z"}`),
	)
	req = req.WithContext(handler.WithUserID(req.Context(), userID))
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/progress/body-weight", s.handler.LogBodyWeight)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	s.Equal(http.StatusCreated, w.Code)
	var resp dto.BodyWeightResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(bwID, resp.ID)
	s.InDelta(80.0, resp.WeightKg, 0.0001)
	s.True(resp.MeasuredAt.Equal(measuredAt))
}

func (s *HandlerTestSuite) TestLogBodyWeight_NoMeasuredAt() {
	userID := uuid.Must(uuid.NewV7())
	s.bodyWeightSvc.EXPECT().
		LogBodyWeight(mock.Anything, mock.MatchedBy(func(cmd servicebodyweight.LogBodyWeightCommand) bool {
			return cmd.UserID == userID && cmd.WeightKg == 80 && cmd.MeasuredAt == nil
		})).
		Return(domainbodyweight.BodyWeight{ID: uuid.Must(uuid.NewV7())}, nil)

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/progress/body-weight",
		strings.NewReader(`{"weight_kg":80}`),
	)
	req = req.WithContext(handler.WithUserID(req.Context(), userID))
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/progress/body-weight", s.handler.LogBodyWeight)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	s.Equal(http.StatusCreated, w.Code)
}

func (s *HandlerTestSuite) TestLogBodyWeight_ValidationError() {
	w := s.serve(http.MethodPost, "/v1/progress/body-weight", `{}`)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("VALIDATION_ERROR", string(resp.Error.Code))
	s.Require().Len(resp.Error.Details, 1)
	s.Equal("weight_kg", resp.Error.Details[0].Field)
	s.bodyWeightSvc.AssertNotCalled(s.T(), "LogBodyWeight", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestLogBodyWeight_InvalidJSON() {
	w := s.serve(http.MethodPost, "/v1/progress/body-weight", `{"weight_kg":`)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("INVALID_REQUEST_BODY", string(resp.Error.Code))
	s.bodyWeightSvc.AssertNotCalled(s.T(), "LogBodyWeight", mock.Anything, mock.Anything)
}

func TestHandlerSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(HandlerTestSuite))
}
