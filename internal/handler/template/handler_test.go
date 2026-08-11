package template_test

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
	"github.com/vladgrskkh/onerep-api/internal/handler"
	"github.com/vladgrskkh/onerep-api/internal/handler/template"
	"github.com/vladgrskkh/onerep-api/internal/handler/template/dto"
	templatemocks "github.com/vladgrskkh/onerep-api/internal/handler/template/mocks"
	servicetemplate "github.com/vladgrskkh/onerep-api/internal/service/template"
)

type HandlerTestSuite struct {
	suite.Suite

	handler *template.TemplateHandler
	svc     *templatemocks.MockTemplateService
	mux     *http.ServeMux
	userID  uuid.UUID
}

func (s *HandlerTestSuite) SetupTest() {
	s.userID = uuid.Must(uuid.NewV7())
	s.svc = templatemocks.NewMockTemplateService(s.T())
	logger := slog.New(slog.DiscardHandler)
	s.handler = template.NewTemplateHandler(s.svc, logger)

	s.mux = http.NewServeMux()
	s.mux.HandleFunc("GET /v1/templates", s.handler.List)
	s.mux.HandleFunc("GET /v1/templates/{id}", s.handler.Get)
	s.mux.HandleFunc("POST /v1/templates", s.handler.Create)
	s.mux.HandleFunc("PATCH /v1/templates/{id}", s.handler.Update)
	s.mux.HandleFunc("POST /v1/templates/{id}/publish", s.handler.Publish)
	s.mux.HandleFunc("POST /v1/templates/{id}/fork", s.handler.Fork)
	s.mux.HandleFunc("DELETE /v1/templates/{id}", s.handler.SoftDelete)
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

func (s *HandlerTestSuite) TestList_Success() {
	since := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	templates := []*domaintemplate.Template{{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "Push Day",
	}}
	s.svc.EXPECT().List(mock.Anything, domaintemplate.TemplateFilter{
		UserID: &s.userID,
		Since:  &since,
	}).Return(templates, nil)

	w := s.serveAs(
		http.MethodGet,
		"/v1/templates?since=2026-08-01T00%3A00%3A00Z",
		"",
		s.userID,
	)

	s.Equal(http.StatusOK, w.Code)
	var resp []dto.TemplateResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Require().Len(resp, 1)
	s.Equal(templates[0].ID, resp[0].ID)
	s.Equal("Push Day", resp[0].Name)
}

func (s *HandlerTestSuite) TestList_InvalidSince() {
	w := s.serve(http.MethodGet, "/v1/templates?since=not-a-timestamp", "")

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("INVALID_SINCE", string(resp.Error.Code))
	s.svc.AssertNotCalled(s.T(), "List", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestList_ServiceError() {
	s.svc.EXPECT().List(mock.Anything, mock.Anything).
		Return(nil, domaintemplate.ErrInvalidUserID)

	w := s.serveAs(http.MethodGet, "/v1/templates", "", s.userID)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("INVALID_USER_ID", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestGet_Success() {
	templateID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().Get(mock.Anything, templateID, s.userID).
		Return(&domaintemplate.Template{ID: templateID, Name: "Push Day", CreatedByUserID: s.userID}, nil)

	w := s.serveAs(http.MethodGet, "/v1/templates/"+templateID.String(), "", s.userID)

	s.Equal(http.StatusOK, w.Code)
	var resp dto.TemplateResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(templateID, resp.ID)
	s.Equal("Push Day", resp.Name)
}

func (s *HandlerTestSuite) TestGet_PrivateByOther() {
	templateID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().Get(mock.Anything, templateID, userID).
		Return(nil, domaintemplate.ErrNotOwner)

	w := s.serveAs(http.MethodGet, "/v1/templates/"+templateID.String(), "", userID)

	s.Equal(http.StatusForbidden, w.Code)
	resp := s.decodeError(w)
	s.Equal("FORBIDDEN", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestGet_InvalidID() {
	w := s.serve(http.MethodGet, "/v1/templates/not-a-uuid", "")

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("INVALID_TEMPLATE_ID", string(resp.Error.Code))
	s.svc.AssertNotCalled(s.T(), "Get", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestGet_NotFound() {
	templateID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().Get(mock.Anything, templateID, mock.Anything).
		Return(nil, domaintemplate.ErrTemplateNotFound)

	w := s.serveAs(http.MethodGet, "/v1/templates/"+templateID.String(), "", s.userID)

	s.Equal(http.StatusNotFound, w.Code)
	resp := s.decodeError(w)
	s.Equal("TEMPLATE_NOT_FOUND", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestCreate_ValidationError() {
	w := s.serve(http.MethodPost, "/v1/templates", `{"name":""}`)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("VALIDATION_ERROR", string(resp.Error.Code))
	s.Require().Len(resp.Error.Details, 1)
	s.Equal("name", resp.Error.Details[0].Field)
	s.Equal("required", resp.Error.Details[0].Tag)
	s.svc.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestCreate_InvalidExerciseID() {
	w := s.serve(http.MethodPost, "/v1/templates", `{"name":"Push Day","exercises":[{"planned_sets":3}]}`)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("VALIDATION_ERROR", string(resp.Error.Code))
	s.Require().Len(resp.Error.Details, 1)
	s.Equal("exercise_id", resp.Error.Details[0].Field)
	s.Equal("required", resp.Error.Details[0].Tag)
	s.svc.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestCreate_InvalidJSON() {
	w := s.serve(http.MethodPost, "/v1/templates", `{"name":`)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("INVALID_REQUEST_BODY", string(resp.Error.Code))
	s.svc.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestCreate_Success() {
	templateID := uuid.Must(uuid.NewV7())
	exerciseID := uuid.Must(uuid.NewV7())
	created := &domaintemplate.Template{
		ID:              templateID,
		Name:            "Push Day",
		CreatedByUserID: s.userID,
		Exercises: []domaintemplate.TemplateExercise{
			{TemplateID: templateID, ExerciseID: exerciseID, SortOrder: 1, PlannedSets: 3},
		},
	}
	s.svc.EXPECT().Create(mock.Anything, servicetemplate.CreateTemplateCommand{
		Name:        "Push Day",
		Description: "Chest, shoulders, triceps",
		UserID:      s.userID,
		Exercises: []servicetemplate.TemplateExerciseCommand{
			{ExerciseID: exerciseID, PlannedSets: 3},
		},
	}).Return(created, nil)

	w := s.serveAs(
		http.MethodPost,
		"/v1/templates",
		`{"name":"Push Day","description":"Chest, shoulders, triceps","exercises":[{"exercise_id":"`+exerciseID.String()+`","planned_sets":3}]}`,
		s.userID,
	)

	s.Equal(http.StatusCreated, w.Code)
	var resp dto.TemplateResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(templateID, resp.ID)
	s.Equal("Push Day", resp.Name)
	s.Equal(s.userID, resp.CreatedByUserID)
	s.Require().Len(resp.Exercises, 1)
	s.Equal(exerciseID, resp.Exercises[0].ExerciseID)
	s.Equal(3, resp.Exercises[0].PlannedSets)
}

func (s *HandlerTestSuite) TestCreate_WithoutUser() {
	templateID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().Create(mock.Anything, mock.Anything).
		Return(&domaintemplate.Template{ID: templateID, Name: "Push Day"}, nil)

	w := s.serve(http.MethodPost, "/v1/templates", `{"name":"Push Day"}`)

	s.Equal(http.StatusCreated, w.Code)
	var resp dto.TemplateResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(templateID, resp.ID)
}

func (s *HandlerTestSuite) TestUpdate_NotOwner() {
	templateID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().Update(mock.Anything, mock.Anything).
		Return(nil, domaintemplate.ErrNotOwner)

	w := s.serveAs(http.MethodPatch, "/v1/templates/"+templateID.String(), `{"name":"Renamed"}`, s.userID)

	s.Equal(http.StatusForbidden, w.Code)
	resp := s.decodeError(w)
	s.Equal("FORBIDDEN", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestUpdate_Success() {
	templateID := uuid.Must(uuid.NewV7())
	newName := "Heavy Push Day"
	updated := &domaintemplate.Template{ID: templateID, Name: newName}
	s.svc.EXPECT().Update(mock.Anything, servicetemplate.UpdateTemplateCommand{
		ID:     templateID,
		UserID: s.userID,
		Name:   &newName,
	}).Return(updated, nil)

	w := s.serveAs(
		http.MethodPatch,
		"/v1/templates/"+templateID.String(),
		`{"name":"Heavy Push Day"}`,
		s.userID,
	)

	s.Equal(http.StatusOK, w.Code)
	var resp dto.TemplateResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(newName, resp.Name)
}

func (s *HandlerTestSuite) TestPublish_Success() {
	templateID := uuid.Must(uuid.NewV7())
	published := &domaintemplate.Template{ID: templateID, Name: "Push Day", IsPublic: true}
	s.svc.EXPECT().Publish(mock.Anything, templateID, s.userID).Return(published, nil)

	w := s.serveAs(http.MethodPost, "/v1/templates/"+templateID.String()+"/publish", "", s.userID)

	s.Equal(http.StatusOK, w.Code)
	var resp dto.TemplateResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.True(resp.IsPublic)
}

func (s *HandlerTestSuite) TestPublish_NotFound() {
	templateID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().Publish(mock.Anything, templateID, mock.Anything).
		Return(nil, domaintemplate.ErrTemplateNotFound)

	w := s.serveAs(http.MethodPost, "/v1/templates/"+templateID.String()+"/publish", "", s.userID)

	s.Equal(http.StatusNotFound, w.Code)
	resp := s.decodeError(w)
	s.Equal("TEMPLATE_NOT_FOUND", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestFork_Success() {
	templateID := uuid.Must(uuid.NewV7())
	forkedID := uuid.Must(uuid.NewV7())
	fork := &domaintemplate.Template{ID: forkedID, Name: "Push Day", CreatedByUserID: s.userID}
	s.svc.EXPECT().Fork(mock.Anything, templateID, s.userID).Return(fork, nil)

	w := s.serveAs(http.MethodPost, "/v1/templates/"+templateID.String()+"/fork", "", s.userID)

	s.Equal(http.StatusCreated, w.Code)
	var resp dto.TemplateResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(forkedID, resp.ID)
}

func (s *HandlerTestSuite) TestFork_PrivateByOther() {
	templateID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().Fork(mock.Anything, templateID, userID).
		Return(nil, domaintemplate.ErrNotOwner)

	w := s.serveAs(http.MethodPost, "/v1/templates/"+templateID.String()+"/fork", "", userID)

	s.Equal(http.StatusForbidden, w.Code)
	resp := s.decodeError(w)
	s.Equal("FORBIDDEN", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestFork_InvalidID() {
	w := s.serve(http.MethodPost, "/v1/templates/not-a-uuid/fork", "")

	s.Equal(http.StatusBadRequest, w.Code)
	resp := s.decodeError(w)
	s.Equal("INVALID_TEMPLATE_ID", string(resp.Error.Code))
	s.svc.AssertNotCalled(s.T(), "Fork", mock.Anything, mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestSoftDelete_Success() {
	templateID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().SoftDelete(mock.Anything, templateID, s.userID).Return(nil)

	w := s.serveAs(http.MethodDelete, "/v1/templates/"+templateID.String(), "", s.userID)

	s.Equal(http.StatusNoContent, w.Code)
}

func TestHandlerSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(HandlerTestSuite))
}
