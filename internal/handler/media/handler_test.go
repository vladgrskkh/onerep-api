package media_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	domainexercise "github.com/vladgrskkh/onerep-api/internal/domain/exercise"
	domaintemplate "github.com/vladgrskkh/onerep-api/internal/domain/template"
	"github.com/vladgrskkh/onerep-api/internal/handler"
	"github.com/vladgrskkh/onerep-api/internal/handler/exercise/dto"
	"github.com/vladgrskkh/onerep-api/internal/handler/media"
	mediamocks "github.com/vladgrskkh/onerep-api/internal/handler/media/mocks"
	templatedto "github.com/vladgrskkh/onerep-api/internal/handler/template/dto"
	"github.com/vladgrskkh/onerep-api/internal/handler/testutil"
	serviceexercise "github.com/vladgrskkh/onerep-api/internal/service/exercise"
	servicetemplate "github.com/vladgrskkh/onerep-api/internal/service/template"
)

// oversizeBytes is the smallest payload that exceeds the handler upload
// limit and must be rejected with 413.
const oversizeBytes = 10<<20 + 1

type HandlerTestSuite struct {
	suite.Suite

	handler     *media.MediaHandler
	storage     *mediamocks.MockObjectStorage
	exerciseSvc *mediamocks.MockExerciseService
	templateSvc *mediamocks.MockTemplateService
	router      *chi.Mux
}

func (s *HandlerTestSuite) SetupTest() {
	s.storage = mediamocks.NewMockObjectStorage(s.T())
	s.exerciseSvc = mediamocks.NewMockExerciseService(s.T())
	s.templateSvc = mediamocks.NewMockTemplateService(s.T())
	s.handler = media.NewMediaHandler(s.storage, s.exerciseSvc, s.templateSvc, slog.New(slog.DiscardHandler))

	s.router = chi.NewRouter()
	s.router.Post("/v1/exercises/{id}/media", s.handler.UploadExerciseMedia)
	s.router.Post("/v1/templates/{id}/media", s.handler.UploadTemplateMedia)
}

// serveMultipart builds a multipart request with the given file part and
// media type field and serves it on the router.
func (s *HandlerTestSuite) serveMultipart(
	target, mediaType, contentType string,
	payload []byte,
	userID uuid.UUID,
) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if mediaType != "" {
		s.Require().NoError(w.WriteField("media_type", mediaType))
	}
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="upload.bin"`)
	header.Set("Content-Type", contentType)
	part, err := w.CreatePart(header)
	s.Require().NoError(err)
	_, err = part.Write(payload)
	s.Require().NoError(err)
	s.Require().NoError(w.Close())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, target, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if userID != uuid.Nil {
		req = req.WithContext(handler.WithUserID(req.Context(), userID))
	}
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)
	return rec
}

func (s *HandlerTestSuite) TestUploadExerciseMedia_Success() {
	exerciseID := uuid.Must(uuid.NewV7())
	key := "exercises/" + exerciseID.String() + "/media-uuid"
	media := &domainexercise.ExerciseMedia{
		ID:         uuid.Must(uuid.NewV7()),
		ExerciseID: exerciseID,
		MediaType:  domainexercise.MediaTypePhoto,
		SortOrder:  0,
		S3Key:      key,
	}
	s.storage.EXPECT().GenerateKey("exercises/" + exerciseID.String()).Return(key)
	s.storage.EXPECT().Upload(mock.Anything, key, mock.Anything, "image/jpeg").Return(nil)
	s.exerciseSvc.EXPECT().UploadMedia(mock.Anything, serviceexercise.UploadExerciseMediaCommand{
		ExerciseID: exerciseID,
		MediaType:  domainexercise.MediaTypePhoto,
		S3Key:      key,
	}).Return(media, nil)

	w := s.serveMultipart(
		"/v1/exercises/"+exerciseID.String()+"/media",
		"photo",
		"image/jpeg",
		[]byte("fake image bytes"),
		uuid.Nil,
	)

	s.Equal(http.StatusCreated, w.Code)
	var resp dto.ExerciseMediaResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(media.ID, resp.ID)
	s.Equal("photo", resp.MediaType)
	s.Equal(key, resp.S3Key)
}

func (s *HandlerTestSuite) TestUploadExerciseMedia_Video() {
	exerciseID := uuid.Must(uuid.NewV7())
	key := "exercises/" + exerciseID.String() + "/media-uuid"
	s.storage.EXPECT().GenerateKey("exercises/" + exerciseID.String()).Return(key)
	s.storage.EXPECT().Upload(mock.Anything, key, mock.Anything, "video/mp4").Return(nil)
	s.exerciseSvc.EXPECT().
		UploadMedia(mock.Anything, mock.MatchedBy(func(cmd serviceexercise.UploadExerciseMediaCommand) bool {
			return cmd.ExerciseID == exerciseID && cmd.MediaType == domainexercise.MediaTypeVideo && cmd.S3Key == key
		})).
		Return(&domainexercise.ExerciseMedia{
			ID:         uuid.Must(uuid.NewV7()),
			ExerciseID: exerciseID,
			MediaType:  domainexercise.MediaTypeVideo,
			S3Key:      key,
		}, nil)

	w := s.serveMultipart(
		"/v1/exercises/"+exerciseID.String()+"/media",
		"video",
		"video/mp4",
		[]byte("fake video bytes"),
		uuid.Nil,
	)

	s.Equal(http.StatusCreated, w.Code)
}

func (s *HandlerTestSuite) TestUploadExerciseMedia_InvalidID() {
	w := s.serveMultipart(
		"/v1/exercises/not-a-uuid/media",
		"photo",
		"image/jpeg",
		[]byte("fake image bytes"),
		uuid.Nil,
	)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INVALID_EXERCISE_ID", string(resp.Error.Code))
	s.storage.AssertNotCalled(s.T(), "GenerateKey", mock.Anything)
}

func (s *HandlerTestSuite) TestUploadExerciseMedia_UnsupportedContentType() {
	w := s.serveMultipart(
		"/v1/exercises/"+uuid.Must(uuid.NewV7()).String()+"/media",
		"photo",
		"text/plain",
		[]byte("hello"),
		uuid.Nil,
	)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("UNSUPPORTED_MEDIA_TYPE", string(resp.Error.Code))
	s.storage.AssertNotCalled(s.T(), "Upload", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestUploadExerciseMedia_ContentTypeMismatch() {
	w := s.serveMultipart(
		"/v1/exercises/"+uuid.Must(uuid.NewV7()).String()+"/media",
		"photo",
		"video/mp4",
		[]byte("video pretending to be a photo"),
		uuid.Nil,
	)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("UNSUPPORTED_MEDIA_TYPE", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestUploadExerciseMedia_InvalidMediaType() {
	w := s.serveMultipart(
		"/v1/exercises/"+uuid.Must(uuid.NewV7()).String()+"/media",
		"gif",
		"image/jpeg",
		[]byte("fake image bytes"),
		uuid.Nil,
	)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INVALID_MEDIA_TYPE", string(resp.Error.Code))
	s.storage.AssertNotCalled(s.T(), "Upload", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestUploadExerciseMedia_MissingFile() {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	s.Require().NoError(w.WriteField("media_type", "photo"))
	s.Require().NoError(w.Close())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost,
		"/v1/exercises/"+uuid.Must(uuid.NewV7()).String()+"/media", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)
	resp := testutil.DecodeError(s.T(), rec)
	s.Equal("INVALID_REQUEST_BODY", string(resp.Error.Code))
	s.storage.AssertNotCalled(s.T(), "Upload", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestUploadExerciseMedia_PayloadTooLarge() {
	w := s.serveMultipart(
		"/v1/exercises/"+uuid.Must(uuid.NewV7()).String()+"/media",
		"photo",
		"image/jpeg",
		bytes.Repeat([]byte{'x'}, oversizeBytes),
		uuid.Nil,
	)

	s.Equal(http.StatusRequestEntityTooLarge, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("PAYLOAD_TOO_LARGE", string(resp.Error.Code))
	s.storage.AssertNotCalled(s.T(), "Upload", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestUploadExerciseMedia_ExerciseNotFound() {
	exerciseID := uuid.Must(uuid.NewV7())
	key := "exercises/" + exerciseID.String() + "/media-uuid"
	s.storage.EXPECT().GenerateKey("exercises/" + exerciseID.String()).Return(key)
	s.storage.EXPECT().Upload(mock.Anything, key, mock.Anything, "image/jpeg").Return(nil)
	s.storage.EXPECT().Delete(mock.Anything, key).Return(nil)
	s.exerciseSvc.EXPECT().UploadMedia(mock.Anything, mock.Anything).
		Return(nil, domainexercise.ErrExerciseNotFound)

	w := s.serveMultipart(
		"/v1/exercises/"+exerciseID.String()+"/media",
		"photo",
		"image/jpeg",
		[]byte("fake image bytes"),
		uuid.Nil,
	)

	s.Equal(http.StatusNotFound, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("EXERCISE_NOT_FOUND", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestUploadExerciseMedia_DeleteFailureKeepsOriginalError() {
	exerciseID := uuid.Must(uuid.NewV7())
	key := "exercises/" + exerciseID.String() + "/media-uuid"
	s.storage.EXPECT().GenerateKey("exercises/" + exerciseID.String()).Return(key)
	s.storage.EXPECT().Upload(mock.Anything, key, mock.Anything, "image/jpeg").Return(nil)
	s.storage.EXPECT().Delete(mock.Anything, key).Return(errors.New("delete failed"))
	s.exerciseSvc.EXPECT().UploadMedia(mock.Anything, mock.Anything).
		Return(nil, domainexercise.ErrExerciseNotFound)

	w := s.serveMultipart(
		"/v1/exercises/"+exerciseID.String()+"/media",
		"photo",
		"image/jpeg",
		[]byte("fake image bytes"),
		uuid.Nil,
	)

	s.Equal(http.StatusNotFound, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("EXERCISE_NOT_FOUND", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestUploadExerciseMedia_BuiltIn() {
	exerciseID := uuid.Must(uuid.NewV7())
	key := "exercises/" + exerciseID.String() + "/media-uuid"
	s.storage.EXPECT().GenerateKey("exercises/" + exerciseID.String()).Return(key)
	s.storage.EXPECT().Upload(mock.Anything, key, mock.Anything, "image/jpeg").Return(nil)
	s.storage.EXPECT().Delete(mock.Anything, key).Return(nil)
	s.exerciseSvc.EXPECT().UploadMedia(mock.Anything, mock.Anything).
		Return(nil, domainexercise.ErrCannotEditBuiltIn)

	w := s.serveMultipart(
		"/v1/exercises/"+exerciseID.String()+"/media",
		"photo",
		"image/jpeg",
		[]byte("fake image bytes"),
		uuid.Nil,
	)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("CANNOT_EDIT_BUILT_IN", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestUploadTemplateMedia_Success() {
	templateID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	key := "templates/" + templateID.String() + "/media-uuid"
	media := &domaintemplate.TemplateMedia{
		ID:         uuid.Must(uuid.NewV7()),
		TemplateID: templateID,
		MediaType:  domaintemplate.MediaTypePhoto,
		SortOrder:  0,
		S3Key:      key,
	}
	s.storage.EXPECT().GenerateKey("templates/" + templateID.String()).Return(key)
	s.storage.EXPECT().Upload(mock.Anything, key, mock.Anything, "image/png").Return(nil)
	s.templateSvc.EXPECT().UploadMedia(mock.Anything, servicetemplate.UploadTemplateMediaCommand{
		TemplateID: templateID,
		UserID:     userID,
		S3Key:      key,
	}).Return(media, nil)

	w := s.serveMultipart(
		"/v1/templates/"+templateID.String()+"/media",
		"",
		"image/png",
		[]byte("fake image bytes"),
		userID,
	)

	s.Equal(http.StatusCreated, w.Code)
	var resp templatedto.TemplateMediaResponse
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal(media.ID, resp.ID)
	s.Equal("photo", resp.MediaType)
	s.Equal(key, resp.S3Key)
}

func (s *HandlerTestSuite) TestUploadTemplateMedia_NotOwner() {
	templateID := uuid.Must(uuid.NewV7())
	key := "templates/" + templateID.String() + "/media-uuid"
	s.storage.EXPECT().GenerateKey("templates/" + templateID.String()).Return(key)
	s.storage.EXPECT().Upload(mock.Anything, key, mock.Anything, "image/jpeg").Return(nil)
	s.storage.EXPECT().Delete(mock.Anything, key).Return(nil)
	s.templateSvc.EXPECT().UploadMedia(mock.Anything, mock.Anything).
		Return(nil, domaintemplate.ErrNotOwner)

	w := s.serveMultipart(
		"/v1/templates/"+templateID.String()+"/media",
		"",
		"image/jpeg",
		[]byte("fake image bytes"),
		uuid.Must(uuid.NewV7()),
	)

	s.Equal(http.StatusForbidden, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("FORBIDDEN", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestUploadTemplateMedia_NotFound() {
	templateID := uuid.Must(uuid.NewV7())
	key := "templates/" + templateID.String() + "/media-uuid"
	s.storage.EXPECT().GenerateKey("templates/" + templateID.String()).Return(key)
	s.storage.EXPECT().Upload(mock.Anything, key, mock.Anything, "image/jpeg").Return(nil)
	s.storage.EXPECT().Delete(mock.Anything, key).Return(nil)
	s.templateSvc.EXPECT().UploadMedia(mock.Anything, mock.Anything).
		Return(nil, domaintemplate.ErrTemplateNotFound)

	w := s.serveMultipart(
		"/v1/templates/"+templateID.String()+"/media",
		"",
		"image/jpeg",
		[]byte("fake image bytes"),
		uuid.Must(uuid.NewV7()),
	)

	s.Equal(http.StatusNotFound, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("TEMPLATE_NOT_FOUND", string(resp.Error.Code))
}

func (s *HandlerTestSuite) TestUploadTemplateMedia_UnsupportedContentType() {
	w := s.serveMultipart(
		"/v1/templates/"+uuid.Must(uuid.NewV7()).String()+"/media",
		"",
		"video/mp4",
		[]byte("template media is photo-only"),
		uuid.Nil,
	)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("UNSUPPORTED_MEDIA_TYPE", string(resp.Error.Code))
	s.storage.AssertNotCalled(s.T(), "Upload", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func (s *HandlerTestSuite) TestUploadTemplateMedia_InvalidID() {
	w := s.serveMultipart(
		"/v1/templates/not-a-uuid/media",
		"",
		"image/jpeg",
		[]byte("fake image bytes"),
		uuid.Nil,
	)

	s.Equal(http.StatusBadRequest, w.Code)
	resp := testutil.DecodeError(s.T(), w)
	s.Equal("INVALID_TEMPLATE_ID", string(resp.Error.Code))
	s.storage.AssertNotCalled(s.T(), "GenerateKey", mock.Anything)
}

func TestHandlerSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
