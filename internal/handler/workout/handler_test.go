package workout_test

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

	domaintemplate "github.com/vladgrskkh/onerep-api/internal/domain/template"
	domainworkout "github.com/vladgrskkh/onerep-api/internal/domain/workout"
	"github.com/vladgrskkh/onerep-api/internal/handler"
	workouthandler "github.com/vladgrskkh/onerep-api/internal/handler/workout"
	"github.com/vladgrskkh/onerep-api/internal/handler/workout/dto"
	workoutmocks "github.com/vladgrskkh/onerep-api/internal/handler/workout/mocks"
	serviceworkout "github.com/vladgrskkh/onerep-api/internal/service/workout"
)

type HandlerTestSuite struct {
	suite.Suite

	handler *workouthandler.WorkoutHandler
	svc     *workoutmocks.MockWorkoutService
	mux     *http.ServeMux
	userID  uuid.UUID
}

func (s *HandlerTestSuite) SetupTest() {
	s.userID = uuid.Must(uuid.NewV7())
	s.svc = workoutmocks.NewMockWorkoutService(s.T())
	logger := slog.New(slog.DiscardHandler)
	s.handler = workouthandler.NewWorkoutHandler(s.svc, logger)

	s.mux = http.NewServeMux()
	s.mux.HandleFunc("POST /v1/workouts", s.handler.Start)
	s.mux.HandleFunc("GET /v1/workouts", s.handler.List)
	s.mux.HandleFunc("GET /v1/workouts/{id}", s.handler.Get)
	s.mux.HandleFunc("POST /v1/workouts/{id}/exercises", s.handler.AddExercise)
	s.mux.HandleFunc("POST /v1/workouts/{id}/exercises/{exId}/sets", s.handler.LogSet)
	s.mux.HandleFunc("PATCH /v1/workouts/{id}/finish", s.handler.Finish)
}

// serve performs a request without an authenticated user context.
func (s *HandlerTestSuite) serve(method, path string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)
	return w
}

// serveAs performs a request authenticated as the given user.
func (s *HandlerTestSuite) serveAs(method, path string, body string, userID uuid.UUID) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req = req.WithContext(handler.WithUserID(req.Context(), userID))
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)
	return w
}

func (s *HandlerTestSuite) decodeError(w *httptest.ResponseRecorder) handler.ErrorResponse {
	var resp handler.ErrorResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	return resp
}

func (s *HandlerTestSuite) TestStart_Success() {
	templateID := uuid.Must(uuid.NewV7())
	workoutID := uuid.Must(uuid.NewV7())
	created := &domainworkout.Workout{ID: workoutID, UserID: s.userID, TemplateID: &templateID}
	s.svc.EXPECT().Start(mock.Anything, serviceworkout.StartWorkoutCommand{
		UserID:     s.userID,
		TemplateID: &templateID,
	}).Return(created, nil)

	w := s.serveAs(
		http.MethodPost,
		"/v1/workouts",
		`{"template_id":"`+templateID.String()+`"}`,
		s.userID,
	)

	s.Equal(http.StatusCreated, w.Code)
	var resp dto.WorkoutResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(workoutID, resp.ID)
	s.Equal(s.userID, resp.UserID)
	s.Equal(templateID, resp.TemplateID)
}

func (s *HandlerTestSuite) TestStart_EmptyBody() {
	workoutID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().Start(mock.Anything, serviceworkout.StartWorkoutCommand{UserID: s.userID}).
		Return(&domainworkout.Workout{ID: workoutID, UserID: s.userID}, nil)

	w := s.serveAs(http.MethodPost, "/v1/workouts", "", s.userID)

	s.Equal(http.StatusCreated, w.Code)
	var resp dto.WorkoutResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(workoutID, resp.ID)
}

func (s *HandlerTestSuite) TestStart_EmptyJSONObject() {
	workoutID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().Start(mock.Anything, serviceworkout.StartWorkoutCommand{UserID: s.userID}).
		Return(&domainworkout.Workout{ID: workoutID, UserID: s.userID}, nil)

	w := s.serveAs(http.MethodPost, "/v1/workouts", `{}`, s.userID)

	s.Equal(http.StatusCreated, w.Code)
}

func (s *HandlerTestSuite) TestStart_InvalidJSON() {
	w := s.serve(http.MethodPost, "/v1/workouts", `{"template_id":`)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("INVALID_REQUEST_BODY", string(resp.Error.Code))
	s.svc.AssertNotCalled(s.T(), "Start", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestStart_ActiveWorkout() {
	s.svc.EXPECT().Start(mock.Anything, mock.Anything).
		Return(nil, domainworkout.ErrActiveWorkout)

	w := s.serveAs(http.MethodPost, "/v1/workouts", "", s.userID)

	s.Equal(http.StatusConflict, w.Code)
	resp := s.decodeError(w)
	s.Equal("ACTIVE_WORKOUT_EXISTS", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestStart_TemplateNotFound() {
	s.svc.EXPECT().Start(mock.Anything, mock.Anything).
		Return(nil, domaintemplate.ErrTemplateNotFound)

	w := s.serveAs(http.MethodPost, "/v1/workouts", "", s.userID)

	s.Equal(http.StatusNotFound, w.Code)
	resp := s.decodeError(w)
	s.Equal("TEMPLATE_NOT_FOUND", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestList_Success() {
	since := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	workoutID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().List(mock.Anything, s.userID, &since).
		Return([]*domainworkout.Workout{{ID: workoutID, UserID: s.userID}}, nil)

	w := s.serveAs(
		http.MethodGet,
		"/v1/workouts?since=2026-08-01T00%3A00%3A00Z",
		"",
		s.userID,
	)

	s.Equal(http.StatusOK, w.Code)
	var resp []dto.WorkoutResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Require().Len(resp, 1)
	s.Equal(workoutID, resp[0].ID)
}

func (s *HandlerTestSuite) TestList_InvalidSince() {
	w := s.serve(http.MethodGet, "/v1/workouts?since=not-a-timestamp", "")

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("INVALID_SINCE", string(resp.Error.Code))
	s.svc.AssertNotCalled(s.T(), "List", mock.Anything, mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestGet_Success() {
	workoutID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().Get(mock.Anything, workoutID, s.userID).
		Return(&domainworkout.Workout{ID: workoutID, UserID: s.userID}, nil)

	w := s.serveAs(http.MethodGet, "/v1/workouts/"+workoutID.String(), "", s.userID)

	s.Equal(http.StatusOK, w.Code)
	var resp dto.WorkoutResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(workoutID, resp.ID)
}

func (s *HandlerTestSuite) TestGet_InvalidID() {
	w := s.serve(http.MethodGet, "/v1/workouts/not-a-uuid", "")

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("INVALID_WORKOUT_ID", string(resp.Error.Code))
	s.svc.AssertNotCalled(s.T(), "Get", mock.Anything, mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestGet_NotFound() {
	workoutID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().Get(mock.Anything, workoutID, s.userID).
		Return(nil, domainworkout.ErrWorkoutNotFound)

	w := s.serveAs(http.MethodGet, "/v1/workouts/"+workoutID.String(), "", s.userID)

	s.Equal(http.StatusNotFound, w.Code)
	resp := s.decodeError(w)
	s.Equal("WORKOUT_NOT_FOUND", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestGet_NotOwner() {
	workoutID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().Get(mock.Anything, workoutID, s.userID).
		Return(nil, domainworkout.ErrWorkoutNotFound)

	w := s.serveAs(http.MethodGet, "/v1/workouts/"+workoutID.String(), "", s.userID)

	s.Equal(http.StatusNotFound, w.Code)
}

func (s *HandlerTestSuite) TestAddExercise_Success() {
	workoutID := uuid.Must(uuid.NewV7())
	exerciseID := uuid.Must(uuid.NewV7())
	weID := uuid.Must(uuid.NewV7())
	created := &domainworkout.WorkoutExercise{
		ID:         weID,
		WorkoutID:  workoutID,
		ExerciseID: exerciseID,
		SortOrder:  1,
	}
	s.svc.EXPECT().AddExercise(mock.Anything, serviceworkout.AddExerciseCommand{
		WorkoutID:  workoutID,
		ExerciseID: exerciseID,
	}, s.userID).Return(created, nil)

	w := s.serveAs(
		http.MethodPost,
		"/v1/workouts/"+workoutID.String()+"/exercises",
		`{"exercise_id":"`+exerciseID.String()+`"}`,
		s.userID,
	)

	s.Equal(http.StatusCreated, w.Code)
	var resp dto.WorkoutExerciseResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(weID, resp.ID)
	s.Equal(exerciseID, resp.ExerciseID)
	s.Equal(1, resp.SortOrder)
}

func (s *HandlerTestSuite) TestAddExercise_ValidationError() {
	w := s.serve(http.MethodPost, "/v1/workouts/"+uuid.Must(uuid.NewV7()).String()+"/exercises", `{}`)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("VALIDATION_ERROR", string(resp.Error.Code))
	s.Require().Len(resp.Error.Details, 1)
	s.Equal("exercise_id", resp.Error.Details[0].Field)
	s.svc.AssertNotCalled(s.T(), "AddExercise", mock.Anything, mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestAddExercise_InvalidWorkoutID() {
	w := s.serve(
		http.MethodPost,
		"/v1/workouts/not-a-uuid/exercises",
		`{"exercise_id":"`+uuid.Must(uuid.NewV7()).String()+`"}`,
	)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("INVALID_WORKOUT_ID", string(resp.Error.Code))
	s.svc.AssertNotCalled(s.T(), "AddExercise", mock.Anything, mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestLogSet_Success() {
	workoutID := uuid.Must(uuid.NewV7())
	weID := uuid.Must(uuid.NewV7())
	setID := uuid.Must(uuid.NewV7())
	rpe := 8
	result := serviceworkout.SetResult{
		WorkoutSet: domainworkout.WorkoutSet{
			ID:                setID,
			WorkoutExerciseID: weID,
			SetNumber:         1,
			WeightKg:          100,
			Reps:              5,
			RPE:               &rpe,
		},
		IsPR:         true,
		Estimated1RM: 116.7,
	}
	s.svc.EXPECT().LogSet(mock.Anything, serviceworkout.LogSetCommand{
		WorkoutID:         workoutID,
		WorkoutExerciseID: weID,
		WeightKg:          100,
		Reps:              5,
		RPE:               &rpe,
		IsWarmup:          true,
	}, s.userID).Return(result, nil)

	w := s.serveAs(
		http.MethodPost,
		"/v1/workouts/"+workoutID.String()+"/exercises/"+weID.String()+"/sets",
		`{"weight_kg":100,"reps":5,"rpe":8,"is_warmup":true}`,
		s.userID,
	)

	s.Equal(http.StatusCreated, w.Code)
	var resp dto.LogSetResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(setID, resp.ID)
	s.Equal(1, resp.SetNumber)
	s.InDelta(100.0, resp.WeightKg, 0.0001)
	s.Equal(5, resp.Reps)
	s.True(resp.IsPR)
	s.InDelta(116.7, resp.Estimated1RM, 0.0001)
	s.Equal(8, resp.RPE)
}

func (s *HandlerTestSuite) TestLogSet_ValidationError() {
	workoutID := uuid.Must(uuid.NewV7())
	w := s.serve(
		http.MethodPost,
		"/v1/workouts/"+workoutID.String()+"/exercises/"+uuid.Must(uuid.NewV7()).String()+"/sets",
		`{"weight_kg":100}`,
	)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("VALIDATION_ERROR", string(resp.Error.Code))
	s.Require().Len(resp.Error.Details, 1)
	s.Equal("reps", resp.Error.Details[0].Field)
	s.svc.AssertNotCalled(s.T(), "LogSet", mock.Anything, mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestLogSet_InvalidWorkoutExerciseID() {
	w := s.serve(
		http.MethodPost,
		"/v1/workouts/"+uuid.Must(uuid.NewV7()).String()+"/exercises/not-a-uuid/sets",
		`{"weight_kg":100,"reps":5}`,
	)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("INVALID_WORKOUT_EXERCISE_ID", string(resp.Error.Code))
	s.svc.AssertNotCalled(s.T(), "LogSet", mock.Anything, mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestLogSet_NotFound() {
	workoutID := uuid.Must(uuid.NewV7())
	weID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().LogSet(mock.Anything, mock.Anything, s.userID).
		Return(serviceworkout.SetResult{}, domainworkout.ErrWorkoutNotFound)

	w := s.serveAs(
		http.MethodPost,
		"/v1/workouts/"+workoutID.String()+"/exercises/"+weID.String()+"/sets",
		`{"weight_kg":100,"reps":5}`,
		s.userID,
	)

	s.Equal(http.StatusNotFound, w.Code)
	resp := s.decodeError(w)
	s.Equal("WORKOUT_NOT_FOUND", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestFinish_Success() {
	workoutID := uuid.Must(uuid.NewV7())
	finished := &domainworkout.Workout{ID: workoutID, UserID: s.userID}
	s.svc.EXPECT().Finish(mock.Anything, serviceworkout.FinishWorkoutCommand{WorkoutID: workoutID}, s.userID).
		Return(finished, nil)

	w := s.serveAs(http.MethodPatch, "/v1/workouts/"+workoutID.String()+"/finish", "", s.userID)

	s.Equal(http.StatusOK, w.Code)
	var resp dto.WorkoutResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(workoutID, resp.ID)
}

func (s *HandlerTestSuite) TestFinish_NotFound() {
	workoutID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().Finish(mock.Anything, mock.Anything, s.userID).
		Return(nil, domainworkout.ErrWorkoutNotFound)

	w := s.serveAs(http.MethodPatch, "/v1/workouts/"+workoutID.String()+"/finish", "", s.userID)

	s.Equal(http.StatusNotFound, w.Code)
	resp := s.decodeError(w)
	s.Equal("WORKOUT_NOT_FOUND", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestFinish_InvalidID() {
	w := s.serve(http.MethodPatch, "/v1/workouts/not-a-uuid/finish", "")

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("INVALID_WORKOUT_ID", string(resp.Error.Code))
	s.svc.AssertNotCalled(s.T(), "Finish", mock.Anything, mock.Anything, mock.Anything)
}

func TestHandlerSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(HandlerTestSuite))
}
