package exercise_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	domainexercise "github.com/vladgrskkh/onerep-api/internal/domain/exercise"
	"github.com/vladgrskkh/onerep-api/internal/handler/exercise"
	"github.com/vladgrskkh/onerep-api/internal/handler/exercise/dto"
	exercisemocks "github.com/vladgrskkh/onerep-api/internal/handler/exercise/mocks"
	"github.com/vladgrskkh/onerep-api/internal/handler/testutil"
	serviceexercise "github.com/vladgrskkh/onerep-api/internal/service/exercise"
)

type HandlerTestSuite struct {
	suite.Suite

	handler *exercise.ExerciseHandler
	svc     *exercisemocks.MockExerciseService
	userID  uuid.UUID
}

func (s *HandlerTestSuite) SetupTest() {
	s.userID = uuid.Must(uuid.NewV7())
	s.svc = exercisemocks.NewMockExerciseService(s.T())
	s.handler = exercise.NewExerciseHandler(s.svc, slog.New(slog.DiscardHandler))
}

func (s *HandlerTestSuite) listExercises(target string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	s.handler.List(w, testutil.NewRequest(http.MethodGet, target, ""))
	return w
}

func (s *HandlerTestSuite) getExercise(id string) *httptest.ResponseRecorder {
	req := testutil.NewRequest(http.MethodGet, "/v1/exercises/"+id, "")
	req.SetPathValue("id", id)
	w := httptest.NewRecorder()
	s.handler.Get(w, req)
	return w
}

func (s *HandlerTestSuite) createExercise(body string, userID uuid.UUID) *httptest.ResponseRecorder {
	req := testutil.NewRequest(http.MethodPost, "/v1/exercises", body)
	if userID != uuid.Nil {
		req = testutil.AsUser(req, userID)
	}
	w := httptest.NewRecorder()
	s.handler.Create(w, req)
	return w
}

func (s *HandlerTestSuite) updateExercise(id, body string) *httptest.ResponseRecorder {
	req := testutil.NewRequest(http.MethodPatch, "/v1/exercises/"+id, body)
	req.SetPathValue("id", id)
	w := httptest.NewRecorder()
	s.handler.Update(w, req)
	return w
}

func (s *HandlerTestSuite) softDeleteExercise(id string) *httptest.ResponseRecorder {
	req := testutil.NewRequest(http.MethodDelete, "/v1/exercises/"+id, "")
	req.SetPathValue("id", id)
	w := httptest.NewRecorder()
	s.handler.SoftDelete(w, req)
	return w
}

func (s *HandlerTestSuite) TestList_Success() {
	since := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	exercises := []*domainexercise.Exercise{{
		ID:    uuid.Must(uuid.NewV7()),
		Name:  "Bench Press",
		Notes: "Medium grip",
	}}
	s.svc.EXPECT().List(mock.Anything, serviceexercise.ListExercisesCommand{
		Search:      "bench",
		MuscleGroup: "chest",
		Since:       since,
	}).Return(exercises, nil)

	w := s.listExercises("/v1/exercises?search=bench&muscle_group=chest&since=2026-08-01T00%3A00%3A00Z")

	s.Equal(http.StatusOK, w.Code)
	var resp []dto.ExerciseResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Require().Len(resp, 1)
	s.Equal(exercises[0].ID, resp[0].ID)
	s.Equal("Bench Press", resp[0].Name)
}

func (s *HandlerTestSuite) TestList_InvalidSince() {
	w := s.listExercises("/v1/exercises?since=not-a-timestamp")

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INVALID_SINCE", string(resp.Error.Code))
	s.svc.AssertNotCalled(s.T(), "List", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestGet_Success() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().Get(mock.Anything, exerciseID).
		Return(&domainexercise.Exercise{ID: exerciseID, Name: "Squat"}, nil)

	w := s.getExercise(exerciseID.String())

	s.Equal(http.StatusOK, w.Code)
	var resp dto.ExerciseResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(exerciseID, resp.ID)
	s.Equal("Squat", resp.Name)
}

func (s *HandlerTestSuite) TestGet_InvalidID() {
	w := s.getExercise("not-a-uuid")

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INVALID_EXERCISE_ID", string(resp.Error.Code))
	s.svc.AssertNotCalled(s.T(), "Get", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestGet_NotFound() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().Get(mock.Anything, exerciseID).
		Return(nil, domainexercise.ErrExerciseNotFound)

	w := s.getExercise(exerciseID.String())

	s.Equal(http.StatusNotFound, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("EXERCISE_NOT_FOUND", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestCreate_ValidationError() {
	w := s.createExercise(`{"name":""}`, s.userID)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("VALIDATION_ERROR", string(resp.Error.Code))
	s.Require().Len(resp.Error.Details, 1)
	s.Equal("name", resp.Error.Details[0].Field)
	s.Equal("required", resp.Error.Details[0].Tag)
	s.svc.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestCreate_InvalidMuscleGroupID() {
	w := s.createExercise(`{"name":"Bench Press","muscle_group_ids":[0]}`, s.userID)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("VALIDATION_ERROR", string(resp.Error.Code))
	s.Require().Len(resp.Error.Details, 1)
	s.Equal("muscle_group_ids[0]", resp.Error.Details[0].Field)
	s.Equal("gt", resp.Error.Details[0].Tag)
	s.svc.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestCreate_InvalidJSON() {
	w := s.createExercise(`{"name":`, s.userID)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INVALID_REQUEST_BODY", string(resp.Error.Code))
	s.svc.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestCreate_Success() {
	userID := uuid.Must(uuid.NewV7())
	exerciseID := uuid.Must(uuid.NewV7())
	created := &domainexercise.Exercise{
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

	w := s.createExercise(`{"name":"Bench Press","description":"Chest press","muscle_group_ids":[1]}`, userID)

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
	updated := &domainexercise.Exercise{ID: exerciseID, Name: newName}
	s.svc.EXPECT().Update(mock.Anything, serviceexercise.UpdateExerciseCommand{
		ID:   exerciseID,
		Name: &newName,
	}).Return(updated, nil)

	w := s.updateExercise(exerciseID.String(), `{"name":"Incline Bench Press"}`)

	s.Equal(http.StatusOK, w.Code)
	var resp dto.ExerciseResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(newName, resp.Name)
}

func (s *HandlerTestSuite) TestSoftDelete_Success() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().SoftDelete(mock.Anything, exerciseID).Return(nil)

	w := s.softDeleteExercise(exerciseID.String())

	s.Equal(http.StatusNoContent, w.Code)
}

func (s *HandlerTestSuite) TestSoftDelete_BuiltInRejected() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().SoftDelete(mock.Anything, exerciseID).
		Return(domainexercise.ErrCannotEditBuiltIn)

	w := s.softDeleteExercise(exerciseID.String())

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("CANNOT_EDIT_BUILT_IN", string(resp.Error.Code))
}

func TestHandlerSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(HandlerTestSuite))
}
