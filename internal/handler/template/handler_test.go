package template_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	domaintemplate "github.com/vladgrskkh/onerep-api/internal/domain/template"
	"github.com/vladgrskkh/onerep-api/internal/handler/template"
	"github.com/vladgrskkh/onerep-api/internal/handler/template/dto"
	templatemocks "github.com/vladgrskkh/onerep-api/internal/handler/template/mocks"
	"github.com/vladgrskkh/onerep-api/internal/handler/testutil"
	servicetemplate "github.com/vladgrskkh/onerep-api/internal/service/template"
)

const (
	templatesPattern       = "/v1/templates"
	templateByIDPattern    = "/v1/templates/{id}"
	templatePublishPattern = "/v1/templates/{id}/publish"
	templateForkPattern    = "/v1/templates/{id}/fork"
)

type HandlerTestSuite struct {
	suite.Suite

	handler *template.TemplateHandler
	svc     *templatemocks.MockTemplateService
	router  *chi.Mux
	userID  uuid.UUID
}

func (s *HandlerTestSuite) SetupTest() {
	s.userID = uuid.Must(uuid.NewV7())
	s.svc = templatemocks.NewMockTemplateService(s.T())
	s.handler = template.NewTemplateHandler(s.svc, slog.New(slog.DiscardHandler))

	s.router = chi.NewRouter()
	s.router.Get(templatesPattern, s.handler.List)
	s.router.Get(templateByIDPattern, s.handler.Get)
	s.router.Post(templatesPattern, s.handler.Create)
	s.router.Patch(templateByIDPattern, s.handler.Update)
	s.router.Post(templatePublishPattern, s.handler.Publish)
	s.router.Post(templateForkPattern, s.handler.Fork)
	s.router.Delete(templateByIDPattern, s.handler.SoftDelete)
}

func (s *HandlerTestSuite) listTemplates(
	query string,
	userID uuid.UUID,
) *httptest.ResponseRecorder {
	return testutil.Serve(
		s.router,
		http.MethodGet,
		testutil.Path(templatesPattern)+query,
		"",
		userID,
	)
}

func (s *HandlerTestSuite) getTemplate(id string, userID uuid.UUID) *httptest.ResponseRecorder {
	return testutil.Serve(
		s.router,
		http.MethodGet,
		testutil.Path(templateByIDPattern, id),
		"",
		userID,
	)
}

func (s *HandlerTestSuite) createTemplate(
	body string,
	userID uuid.UUID,
) *httptest.ResponseRecorder {
	return testutil.Serve(
		s.router,
		http.MethodPost,
		testutil.Path(templatesPattern),
		body,
		userID,
	)
}

func (s *HandlerTestSuite) updateTemplate(
	id, body string,
	userID uuid.UUID,
) *httptest.ResponseRecorder {
	return testutil.Serve(
		s.router,
		http.MethodPatch,
		testutil.Path(templateByIDPattern, id),
		body,
		userID,
	)
}

func (s *HandlerTestSuite) publishTemplate(id string, userID uuid.UUID) *httptest.ResponseRecorder {
	return testutil.Serve(
		s.router,
		http.MethodPost,
		testutil.Path(templatePublishPattern, id),
		"",
		userID,
	)
}

func (s *HandlerTestSuite) forkTemplate(id string, userID uuid.UUID) *httptest.ResponseRecorder {
	return testutil.Serve(
		s.router,
		http.MethodPost,
		testutil.Path(templateForkPattern, id),
		"",
		userID,
	)
}

func (s *HandlerTestSuite) softDeleteTemplate(
	id string,
	userID uuid.UUID,
) *httptest.ResponseRecorder {
	return testutil.Serve(
		s.router,
		http.MethodDelete,
		testutil.Path(templateByIDPattern, id),
		"",
		userID,
	)
}

func (s *HandlerTestSuite) TestList_Success() {
	since := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	templates := []*domaintemplate.Template{{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "Push Day",
	}}
	s.svc.EXPECT().List(mock.Anything, domaintemplate.TemplateFilter{
		UserID: s.userID,
		Since:  since,
	}).Return(templates, nil)

	w := s.listTemplates("?since=2026-08-01T00%3A00%3A00Z", s.userID)

	s.Equal(http.StatusOK, w.Code)
	var resp []dto.TemplateResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Require().Len(resp, 1)
	s.Equal(templates[0].ID, resp[0].ID)
	s.Equal("Push Day", resp[0].Name)
}

func (s *HandlerTestSuite) TestList_Public() {
	templates := []*domaintemplate.Template{{
		ID:       uuid.Must(uuid.NewV7()),
		Name:     "Shared Push Day",
		IsPublic: true,
	}}
	isPublic := true
	s.svc.EXPECT().List(mock.Anything, domaintemplate.TemplateFilter{
		IsPublic: &isPublic,
	}).Return(templates, nil)

	w := s.listTemplates("?public=true", s.userID)

	s.Equal(http.StatusOK, w.Code)
	var resp []dto.TemplateResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Require().Len(resp, 1)
	s.Equal(templates[0].ID, resp[0].ID)
}

func (s *HandlerTestSuite) TestList_InvalidPublic() {
	w := s.listTemplates("?public=maybe", s.userID)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INVALID_PUBLIC", string(resp.Error.Code))
	s.svc.AssertNotCalled(s.T(), "List", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestList_InvalidSince() {
	w := s.listTemplates("?since=not-a-timestamp", s.userID)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INVALID_SINCE", string(resp.Error.Code))
	s.svc.AssertNotCalled(s.T(), "List", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestList_ServiceError() {
	s.svc.EXPECT().List(mock.Anything, mock.Anything).
		Return(nil, domaintemplate.ErrInvalidUserID)

	w := s.listTemplates("", s.userID)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INVALID_USER_ID", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestGet_Success() {
	templateID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().Get(mock.Anything, templateID, s.userID).
		Return(&domaintemplate.Template{ID: templateID, Name: "Push Day", CreatedByUserID: s.userID}, nil)

	w := s.getTemplate(templateID.String(), s.userID)

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

	w := s.getTemplate(templateID.String(), userID)

	s.Equal(http.StatusForbidden, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("FORBIDDEN", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestGet_InvalidID() {
	w := s.getTemplate("not-a-uuid", s.userID)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INVALID_TEMPLATE_ID", string(resp.Error.Code))
	s.svc.AssertNotCalled(s.T(), "Get", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestGet_NotFound() {
	templateID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().Get(mock.Anything, templateID, mock.Anything).
		Return(nil, domaintemplate.ErrTemplateNotFound)

	w := s.getTemplate(templateID.String(), s.userID)

	s.Equal(http.StatusNotFound, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("TEMPLATE_NOT_FOUND", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestCreate_ValidationError() {
	w := s.createTemplate(`{"name":""}`, s.userID)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("VALIDATION_ERROR", string(resp.Error.Code))
	s.Require().Len(resp.Error.Details, 1)
	s.Equal("name", resp.Error.Details[0].Field)
	s.Equal("required", resp.Error.Details[0].Tag)
	s.svc.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestCreate_InvalidExerciseID() {
	w := s.createTemplate(`{"name":"Push Day","exercises":[{"planned_sets":3}]}`, s.userID)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("VALIDATION_ERROR", string(resp.Error.Code))
	s.Require().Len(resp.Error.Details, 1)
	s.Equal("exercise_id", resp.Error.Details[0].Field)
	s.Equal("required", resp.Error.Details[0].Tag)
	s.svc.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestCreate_InvalidJSON() {
	w := s.createTemplate(`{"name":`, s.userID)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
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

	w := s.createTemplate(
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

	w := s.createTemplate(`{"name":"Push Day"}`, uuid.Nil)

	s.Equal(http.StatusCreated, w.Code)
	var resp dto.TemplateResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(templateID, resp.ID)
}

func (s *HandlerTestSuite) TestUpdate_NotOwner() {
	templateID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().Update(mock.Anything, mock.Anything).
		Return(nil, domaintemplate.ErrNotOwner)

	w := s.updateTemplate(templateID.String(), `{"name":"Renamed"}`, s.userID)

	s.Equal(http.StatusForbidden, w.Code)
	resp := testutil.DecodeError(s.T(), w)
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

	w := s.updateTemplate(templateID.String(), `{"name":"Heavy Push Day"}`, s.userID)

	s.Equal(http.StatusOK, w.Code)
	var resp dto.TemplateResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(newName, resp.Name)
}

func (s *HandlerTestSuite) TestPublish_Success() {
	templateID := uuid.Must(uuid.NewV7())
	published := &domaintemplate.Template{ID: templateID, Name: "Push Day", IsPublic: true}
	s.svc.EXPECT().Publish(mock.Anything, templateID, s.userID).Return(published, nil)

	w := s.publishTemplate(templateID.String(), s.userID)

	s.Equal(http.StatusOK, w.Code)
	var resp dto.TemplateResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.True(resp.IsPublic)
}

func (s *HandlerTestSuite) TestPublish_NotFound() {
	templateID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().Publish(mock.Anything, templateID, mock.Anything).
		Return(nil, domaintemplate.ErrTemplateNotFound)

	w := s.publishTemplate(templateID.String(), s.userID)

	s.Equal(http.StatusNotFound, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("TEMPLATE_NOT_FOUND", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestFork_Success() {
	templateID := uuid.Must(uuid.NewV7())
	forkedID := uuid.Must(uuid.NewV7())
	fork := &domaintemplate.Template{ID: forkedID, Name: "Push Day", CreatedByUserID: s.userID}
	s.svc.EXPECT().Fork(mock.Anything, templateID, s.userID).Return(fork, nil)

	w := s.forkTemplate(templateID.String(), s.userID)

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

	w := s.forkTemplate(templateID.String(), userID)

	s.Equal(http.StatusForbidden, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("FORBIDDEN", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestFork_InvalidID() {
	w := s.forkTemplate("not-a-uuid", s.userID)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INVALID_TEMPLATE_ID", string(resp.Error.Code))
	s.svc.AssertNotCalled(s.T(), "Fork", mock.Anything, mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestSoftDelete_Success() {
	templateID := uuid.Must(uuid.NewV7())
	s.svc.EXPECT().SoftDelete(mock.Anything, templateID, s.userID).Return(nil)

	w := s.softDeleteTemplate(templateID.String(), s.userID)

	s.Equal(http.StatusNoContent, w.Code)
}

func TestHandlerSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(HandlerTestSuite))
}
