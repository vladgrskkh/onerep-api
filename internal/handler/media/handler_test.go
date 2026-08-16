package media_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	domainexercise "github.com/vladgrskkh/onerep-api/internal/domain/exercise"
	domaintemplate "github.com/vladgrskkh/onerep-api/internal/domain/template"
	"github.com/vladgrskkh/onerep-api/internal/handler"
	"github.com/vladgrskkh/onerep-api/internal/handler/media"
	mediadto "github.com/vladgrskkh/onerep-api/internal/handler/media/dto"
	mediamocks "github.com/vladgrskkh/onerep-api/internal/handler/media/mocks"
	"github.com/vladgrskkh/onerep-api/internal/handler/testutil"
	serviceexercise "github.com/vladgrskkh/onerep-api/internal/service/exercise"
	servicetemplate "github.com/vladgrskkh/onerep-api/internal/service/template"
)

type HandlerTestSuite struct {
	suite.Suite

	handler     *media.MediaHandler
	exerciseSvc *mediamocks.MockExerciseService
	templateSvc *mediamocks.MockTemplateService
	router      *chi.Mux
	userID      uuid.UUID
}

func (s *HandlerTestSuite) SetupTest() {
	s.userID = uuid.Must(uuid.NewV7())
	s.exerciseSvc = mediamocks.NewMockExerciseService(s.T())
	s.templateSvc = mediamocks.NewMockTemplateService(s.T())
	s.handler = media.NewMediaHandler(s.exerciseSvc, s.templateSvc, slog.New(slog.DiscardHandler))

	s.router = chi.NewRouter()
	s.router.Post("/v1/exercises/{id}/media", s.handler.UploadExerciseMedia)
	s.router.Post("/v1/templates/{id}/media", s.handler.UploadTemplateMedia)
}

// serveJSON posts a JSON body to the router, optionally as the given user.
func (s *HandlerTestSuite) serveJSON(target, body string, userID uuid.UUID) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		target,
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	if userID != uuid.Nil {
		req = req.WithContext(handler.WithUserID(req.Context(), userID))
	}
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)
	return rec
}

func (s *HandlerTestSuite) TestUploadExerciseMedia_Success() {
	exerciseID := uuid.Must(uuid.NewV7())
	mediaID := uuid.Must(uuid.NewV7())
	s.exerciseSvc.EXPECT().UploadMedia(mock.Anything, serviceexercise.UploadExerciseMediaCommand{
		ExerciseID:  exerciseID,
		UserID:      s.userID,
		MediaType:   domainexercise.MediaTypePhoto,
		ContentType: "image/jpeg",
	}).Return(&serviceexercise.ExerciseMediaUpload{
		Media: &domainexercise.ExerciseMedia{
			ID:         mediaID,
			ExerciseID: exerciseID,
			MediaType:  domainexercise.MediaTypePhoto,
			S3Key:      "exercises/" + exerciseID.String() + "/media-uuid",
		},
		UploadURL: "https://presigned.example/upload",
		ExpiresIn: time.Minute,
	}, nil)

	w := s.serveJSON(
		"/v1/exercises/"+exerciseID.String()+"/media",
		`{"media_type":"photo","content_type":"image/jpeg"}`,
		s.userID,
	)

	s.Equal(http.StatusCreated, w.Code)
	var resp mediadto.MediaUploadResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(mediaID, resp.ID)
	s.Equal("exercises/"+exerciseID.String()+"/media-uuid", resp.S3Key)
	s.Equal("https://presigned.example/upload", resp.UploadURL)
	s.Equal(int64(60), resp.ExpiresIn)
}

func (s *HandlerTestSuite) TestUploadExerciseMedia_Video() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseSvc.EXPECT().
		UploadMedia(mock.Anything, mock.MatchedBy(func(cmd serviceexercise.UploadExerciseMediaCommand) bool {
			return cmd.ExerciseID == exerciseID &&
				cmd.MediaType == domainexercise.MediaTypeVideo &&
				cmd.ContentType == "video/mp4"
		})).
		Return(&serviceexercise.ExerciseMediaUpload{
			Media: &domainexercise.ExerciseMedia{
				ID:         uuid.Must(uuid.NewV7()),
				ExerciseID: exerciseID,
				MediaType:  domainexercise.MediaTypeVideo,
				S3Key:      "exercises/" + exerciseID.String() + "/media-uuid",
			},
			UploadURL: "https://presigned.example/upload",
			ExpiresIn: time.Minute,
		}, nil)

	w := s.serveJSON(
		"/v1/exercises/"+exerciseID.String()+"/media",
		`{"media_type":"video","content_type":"video/mp4"}`,
		uuid.Nil,
	)

	s.Equal(http.StatusCreated, w.Code)
}

func (s *HandlerTestSuite) TestUploadExerciseMedia_InvalidID() {
	w := s.serveJSON(
		"/v1/exercises/not-a-uuid/media",
		`{"media_type":"photo","content_type":"image/jpeg"}`,
		uuid.Nil,
	)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INVALID_EXERCISE_ID", string(resp.Error.Code))
	s.exerciseSvc.AssertNotCalled(s.T(), "UploadMedia", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestUploadExerciseMedia_InvalidMediaType() {
	w := s.serveJSON(
		"/v1/exercises/"+uuid.Must(uuid.NewV7()).String()+"/media",
		`{"media_type":"gif","content_type":"image/jpeg"}`,
		uuid.Nil,
	)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("VALIDATION_ERROR", string(resp.Error.Code))
	s.exerciseSvc.AssertNotCalled(s.T(), "UploadMedia", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestUploadExerciseMedia_MissingContentType() {
	w := s.serveJSON(
		"/v1/exercises/"+uuid.Must(uuid.NewV7()).String()+"/media",
		`{"media_type":"photo"}`,
		uuid.Nil,
	)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("VALIDATION_ERROR", string(resp.Error.Code))
	s.exerciseSvc.AssertNotCalled(s.T(), "UploadMedia", mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestUploadExerciseMedia_UnsupportedContentType() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseSvc.EXPECT().UploadMedia(mock.Anything, mock.Anything).
		Return(nil, domainexercise.ErrUnsupportedContentType)

	w := s.serveJSON(
		"/v1/exercises/"+exerciseID.String()+"/media",
		`{"media_type":"photo","content_type":"text/plain"}`,
		uuid.Nil,
	)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("UNSUPPORTED_MEDIA_TYPE", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestUploadExerciseMedia_InvalidMediaTypeFromService() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseSvc.EXPECT().UploadMedia(mock.Anything, mock.Anything).
		Return(nil, domainexercise.ErrInvalidMediaType)

	w := s.serveJSON(
		"/v1/exercises/"+exerciseID.String()+"/media",
		`{"media_type":"photo","content_type":"image/jpeg"}`,
		uuid.Nil,
	)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INVALID_MEDIA_TYPE", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestUploadExerciseMedia_ExerciseNotFound() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseSvc.EXPECT().UploadMedia(mock.Anything, mock.Anything).
		Return(nil, domainexercise.ErrExerciseNotFound)

	w := s.serveJSON(
		"/v1/exercises/"+exerciseID.String()+"/media",
		`{"media_type":"photo","content_type":"image/jpeg"}`,
		uuid.Nil,
	)

	s.Equal(http.StatusNotFound, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("EXERCISE_NOT_FOUND", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestUploadExerciseMedia_BuiltIn() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseSvc.EXPECT().UploadMedia(mock.Anything, mock.Anything).
		Return(nil, domainexercise.ErrCannotEditBuiltIn)

	w := s.serveJSON(
		"/v1/exercises/"+exerciseID.String()+"/media",
		`{"media_type":"photo","content_type":"image/jpeg"}`,
		uuid.Nil,
	)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("CANNOT_EDIT_BUILT_IN", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestUploadTemplateMedia_Success() {
	templateID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	mediaID := uuid.Must(uuid.NewV7())
	s.templateSvc.EXPECT().UploadMedia(mock.Anything, servicetemplate.UploadTemplateMediaCommand{
		TemplateID:  templateID,
		UserID:      userID,
		ContentType: "image/png",
	}).Return(&servicetemplate.TemplateMediaUpload{
		Media: &domaintemplate.TemplateMedia{
			ID:         mediaID,
			TemplateID: templateID,
			MediaType:  domaintemplate.MediaTypePhoto,
			S3Key:      "templates/" + templateID.String() + "/media-uuid",
		},
		UploadURL: "https://presigned.example/upload",
		ExpiresIn: time.Minute,
	}, nil)

	w := s.serveJSON(
		"/v1/templates/"+templateID.String()+"/media",
		`{"content_type":"image/png"}`,
		userID,
	)

	s.Equal(http.StatusCreated, w.Code)
	var resp mediadto.MediaUploadResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(mediaID, resp.ID)
	s.Equal("templates/"+templateID.String()+"/media-uuid", resp.S3Key)
	s.Equal("https://presigned.example/upload", resp.UploadURL)
	s.Equal(int64(60), resp.ExpiresIn)
}

func (s *HandlerTestSuite) TestUploadTemplateMedia_NotOwner() {
	templateID := uuid.Must(uuid.NewV7())
	s.templateSvc.EXPECT().UploadMedia(mock.Anything, mock.Anything).
		Return(nil, domaintemplate.ErrNotOwner)

	w := s.serveJSON(
		"/v1/templates/"+templateID.String()+"/media",
		`{"content_type":"image/jpeg"}`,
		uuid.Must(uuid.NewV7()),
	)

	s.Equal(http.StatusForbidden, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("FORBIDDEN", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestUploadTemplateMedia_NotFound() {
	templateID := uuid.Must(uuid.NewV7())
	s.templateSvc.EXPECT().UploadMedia(mock.Anything, mock.Anything).
		Return(nil, domaintemplate.ErrTemplateNotFound)

	w := s.serveJSON(
		"/v1/templates/"+templateID.String()+"/media",
		`{"content_type":"image/jpeg"}`,
		uuid.Must(uuid.NewV7()),
	)

	s.Equal(http.StatusNotFound, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("TEMPLATE_NOT_FOUND", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestUploadTemplateMedia_UnsupportedContentType() {
	templateID := uuid.Must(uuid.NewV7())
	s.templateSvc.EXPECT().UploadMedia(mock.Anything, mock.Anything).
		Return(nil, domaintemplate.ErrUnsupportedContentType)

	w := s.serveJSON(
		"/v1/templates/"+templateID.String()+"/media",
		`{"content_type":"video/mp4"}`,
		uuid.Must(uuid.NewV7()),
	)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("UNSUPPORTED_MEDIA_TYPE", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestUploadTemplateMedia_InvalidID() {
	w := s.serveJSON(
		"/v1/templates/not-a-uuid/media",
		`{"content_type":"image/jpeg"}`,
		uuid.Nil,
	)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INVALID_TEMPLATE_ID", string(resp.Error.Code))
	s.templateSvc.AssertNotCalled(s.T(), "UploadMedia", mock.Anything, mock.Anything)
}

func TestHandlerSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
