package exercise_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	domainexercise "github.com/vladgrskkh/onerep-api/internal/domain/exercise"
	"github.com/vladgrskkh/onerep-api/internal/handler"
	"github.com/vladgrskkh/onerep-api/internal/handler/exercise"
	"github.com/vladgrskkh/onerep-api/internal/handler/exercise/dto"
	exercisemocks "github.com/vladgrskkh/onerep-api/internal/handler/exercise/mocks"
	serviceexercise "github.com/vladgrskkh/onerep-api/internal/service/exercise"
)

type HandlerTestSuite struct {
	suite.Suite

	handler *exercise.ExerciseHandler
	svc     *exercisemocks.MockExerciseService
}

func (s *HandlerTestSuite) SetupTest() {
	s.svc = exercisemocks.NewMockExerciseService(s.T())
	logger := slog.New(slog.DiscardHandler)
	s.handler = exercise.NewExerciseHandler(s.svc, logger)
}

func (s *HandlerTestSuite) serve(method, path string, body string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/exercises", s.handler.List)
	mux.HandleFunc("GET /v1/exercises/{id}", s.handler.Get)
	mux.HandleFunc("POST /v1/exercises", s.handler.Create)
	mux.HandleFunc("PATCH /v1/exercises/{id}", s.handler.Update)
	mux.HandleFunc("DELETE /v1/exercises/{id}", s.handler.SoftDelete)

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

func (s *HandlerTestSuite) TestList_Success() {
	since := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	exercises := []domainexercise.Exercise{{
		ID:    uuid.Must(uuid.NewV7()),
		Name:  "Bench Press",
		Notes: "Medium grip",
	}}
	s.svc.EXPECT().List(mock.Anything, serviceexercise.ListExercisesCommand{
		Search:      "bench",
		MuscleGroup: "chest",
		Since:       &since,
	}).Return(exercises, nil)

	w := s.serve(http.MethodGet, "/v1/exercises?search=bench&muscle_group=chest&since=2026-08-01T00%3A00%3A00Z", "")

	s.Equal(http.StatusOK, w.Code)
	var resp []dto.ExerciseResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Require().Len(resp, 1)
	s.Equal(exercises[0].ID, resp[0].ID)
	s.Equal("Bench Press", resp[0].Name)
}

func (s *HandlerTestSuite) TestList_InvalidSince() {
	w := s.serve(http.MethodGet, "/v1/exercises?since=not-a-timestamp", "")

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("INVALID_SINCE", string(resp.Error.Code))
	s.svc.AssertNotCalled(s.T(), "List", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestGet_Success() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().Get(mock.Anything, exerciseID).
		Return(domainexercise.Exercise{ID: exerciseID, Name: "Squat"}, nil)

	w := s.serve(http.MethodGet, "/v1/exercises/"+exerciseID.String(), "")

	s.Equal(http.StatusOK, w.Code)
	var resp dto.ExerciseResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(exerciseID, resp.ID)
	s.Equal("Squat", resp.Name)
}

func (s *HandlerTestSuite) TestGet_InvalidID() {
	w := s.serve(http.MethodGet, "/v1/exercises/not-a-uuid", "")

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("INVALID_EXERCISE_ID", string(resp.Error.Code))
	s.svc.AssertNotCalled(s.T(), "Get", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestGet_NotFound() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().Get(mock.Anything, exerciseID).
		Return(domainexercise.Exercise{}, domainexercise.ErrExerciseNotFound)

	w := s.serve(http.MethodGet, "/v1/exercises/"+exerciseID.String(), "")

	s.Equal(http.StatusNotFound, w.Code)
	resp := s.decodeError(w)
	s.Equal("EXERCISE_NOT_FOUND", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestCreate_ValidationError() {
	w := s.serve(http.MethodPost, "/v1/exercises", `{"name":""}`)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("VALIDATION_ERROR", string(resp.Error.Code))
	s.Require().Len(resp.Error.Details, 1)
	s.Equal("name", resp.Error.Details[0].Field)
	s.Equal("required", resp.Error.Details[0].Tag)
	s.svc.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestCreate_InvalidMuscleGroupID() {
	w := s.serve(http.MethodPost, "/v1/exercises", `{"name":"Bench Press","muscle_group_ids":[0]}`)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("VALIDATION_ERROR", string(resp.Error.Code))
	s.Require().Len(resp.Error.Details, 1)
	s.Equal("muscle_group_ids[0]", resp.Error.Details[0].Field)
	s.Equal("gt", resp.Error.Details[0].Tag)
	s.svc.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestCreate_InvalidJSON() {
	w := s.serve(http.MethodPost, "/v1/exercises", `{"name":`)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("INVALID_REQUEST_BODY", string(resp.Error.Code))
	s.svc.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestCreate_Success() {
	userID := uuid.Must(uuid.NewV7())
	exerciseID := uuid.Must(uuid.NewV7())
	created := domainexercise.Exercise{
		ID:              exerciseID,
		Name:            "Bench Press",
		Description:     "Chest press",
		IsBuiltIn:       false,
		CreatedByUserID: &userID,
		MuscleGroups:    []domainexercise.ExerciseMuscleGroup{{ExerciseID: exerciseID, MuscleGroupID: 1}},
	}
	s.svc.EXPECT().Create(mock.Anything, serviceexercise.CreateExerciseCommand{
		Name:           "Bench Press",
		Description:    "Chest press",
		UserID:         userID,
		MuscleGroupIDs: []int{1},
	}).Return(created, nil)

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/exercises",
		strings.NewReader(`{"name":"Bench Press","description":"Chest press","muscle_group_ids":[1]}`),
	)
	req = req.WithContext(handler.WithUserID(req.Context(), userID))
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/exercises", s.handler.Create)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	s.Equal(http.StatusCreated, w.Code)
	var resp dto.ExerciseResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(exerciseID, resp.ID)
	s.Equal("Bench Press", resp.Name)
	s.Equal(userID, resp.CreatedByUserID)
	s.Require().Len(resp.MuscleGroups, 1)
	s.Equal(1, resp.MuscleGroups[0].ID)
}

func (s *HandlerTestSuite) TestUpdate_Success() {
	exerciseID := uuid.Must(uuid.NewV7())
	newName := "Incline Bench Press"
	updated := domainexercise.Exercise{ID: exerciseID, Name: newName}
	s.svc.EXPECT().Update(mock.Anything, serviceexercise.UpdateExerciseCommand{
		ID:   exerciseID,
		Name: &newName,
	}).Return(updated, nil)

	w := s.serve(http.MethodPatch, "/v1/exercises/"+exerciseID.String(), `{"name":"Incline Bench Press"}`)

	s.Equal(http.StatusOK, w.Code)
	var resp dto.ExerciseResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(newName, resp.Name)
}

func (s *HandlerTestSuite) TestSoftDelete_Success() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().SoftDelete(mock.Anything, exerciseID).Return(nil)

	w := s.serve(http.MethodDelete, "/v1/exercises/"+exerciseID.String(), "")

	s.Equal(http.StatusNoContent, w.Code)
}

func (s *HandlerTestSuite) TestSoftDelete_BuiltInRejected() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().SoftDelete(mock.Anything, exerciseID).
		Return(domainexercise.ErrCannotEditBuiltIn)

	w := s.serve(http.MethodDelete, "/v1/exercises/"+exerciseID.String(), "")

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("CANNOT_EDIT_BUILT_IN", string(resp.Error.Code))
}

func TestHandlerSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(HandlerTestSuite))
}
