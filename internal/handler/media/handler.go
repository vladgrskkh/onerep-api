package media

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"slices"

	"github.com/google/uuid"

	domainexercise "github.com/vladgrskkh/onerep-api/internal/domain/exercise"
	domaintemplate "github.com/vladgrskkh/onerep-api/internal/domain/template"
	"github.com/vladgrskkh/onerep-api/internal/handler"
	serviceexercise "github.com/vladgrskkh/onerep-api/internal/service/exercise"
	servicetemplate "github.com/vladgrskkh/onerep-api/internal/service/template"
)

// maxUploadSize bounds the total size of a single multipart media upload.
const maxUploadSize = 10 << 20 // 10 MiB

// ExerciseService is the exercise media upload use case consumed by the
// media handler.
type ExerciseService interface {
	UploadMedia(
		ctx context.Context,
		cmd serviceexercise.UploadExerciseMediaCommand,
	) (*domainexercise.ExerciseMedia, error)
}

// TemplateService is the template media upload use case consumed by the
// media handler.
type TemplateService interface {
	UploadMedia(
		ctx context.Context,
		cmd servicetemplate.UploadTemplateMediaCommand,
	) (*domaintemplate.TemplateMedia, error)
}

// ObjectStorage uploads and removes media objects in the backing object
// store.
type ObjectStorage interface {
	Upload(ctx context.Context, key string, body io.Reader, contentType string) error
	Delete(ctx context.Context, key string) error
	GenerateKey(prefix string) string
}

// MediaHandler uploads media objects for exercises and templates.
type MediaHandler struct {
	storage     ObjectStorage
	exerciseSvc ExerciseService
	templateSvc TemplateService
	logger      *slog.Logger
}

func NewMediaHandler(
	storage ObjectStorage,
	exerciseSvc ExerciseService,
	templateSvc TemplateService,
	logger *slog.Logger,
) *MediaHandler {
	return &MediaHandler{
		storage:     storage,
		exerciseSvc: exerciseSvc,
		templateSvc: templateSvc,
		logger:      logger,
	}
}

// UploadExerciseMedia stores a photo or video for an exercise.
//
// @Summary Upload exercise media
// @Description Upload a photo or video for an exercise; the file is stored in the media bucket and linked to the exercise
// @Tags exercises
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "Exercise ID"
// @Param file formData file true "Media file (image/jpeg, image/png, image/webp, or video/mp4)"
// @Param media_type formData string true "Media type" Enums(photo, video)
// @Success 201 {object} dto.ExerciseMediaResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 404 {object} handler.ErrorResponse
// @Failure 413 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /exercises/{id}/media [post]
func (h *MediaHandler) UploadExerciseMedia(w http.ResponseWriter, r *http.Request) {
	exerciseID, parseErr := uuid.Parse(r.PathValue("id"))
	if parseErr != nil {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidExerciseIDDetail())
		return
	}

	file, contentType, err := h.parseUpload(w, r)
	if err != nil {
		h.writeUploadParseError(w, err)
		return
	}
	defer file.Close()

	mediaType, err := parseMediaType(r.FormValue("media_type"))
	if err != nil {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidMediaTypeDetail(err))
		return
	}
	if !contentTypeAllowed(mediaType == domainexercise.MediaTypePhoto, contentType) {
		handler.WriteError(w, h.logger, http.StatusBadRequest, unsupportedMediaTypeDetail())
		return
	}

	key := h.storage.GenerateKey("exercises/" + exerciseID.String())
	err = h.storage.Upload(r.Context(), key, file, contentType)
	if err != nil {
		handler.WriteSystemError(w, h.logger, http.StatusInternalServerError, err)
		return
	}

	media, err := h.exerciseSvc.UploadMedia(r.Context(), serviceexercise.UploadExerciseMediaCommand{
		ExerciseID: exerciseID,
		MediaType:  mediaType,
		S3Key:      key,
	})
	if err != nil {
		h.deleteUploadedObject(r.Context(), key)
		switch {
		case errors.Is(err, domainexercise.ErrExerciseNotFound):
			handler.WriteError(w, h.logger, http.StatusNotFound, exerciseNotFoundDetail(err))
		case errors.Is(err, domainexercise.ErrCannotEditBuiltIn):
			handler.WriteError(w, h.logger, http.StatusBadRequest, cannotEditBuiltInDetail(err))
		case errors.Is(err, domainexercise.ErrInvalidMediaType):
			handler.WriteError(w, h.logger, http.StatusBadRequest, invalidMediaTypeDetail(err))
		case errors.Is(err, domainexercise.ErrInvalidS3Key):
			handler.WriteError(w, h.logger, http.StatusBadRequest, invalidRequestBodyDetail())
		default:
			handler.WriteSystemError(w, h.logger, http.StatusInternalServerError, err)
		}
		return
	}

	handler.WriteJSON(w, h.logger, http.StatusCreated, toExerciseMediaResponse(media))
}

// UploadTemplateMedia stores a photo for a template.
//
// @Summary Upload template media
// @Description Upload a photo for a template; the file is stored in the media bucket and linked to the template
// @Tags templates
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "Template ID"
// @Param file formData file true "Media file (image/jpeg, image/png, or image/webp)"
// @Success 201 {object} dto.TemplateMediaResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 403 {object} handler.ErrorResponse
// @Failure 404 {object} handler.ErrorResponse
// @Failure 413 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /templates/{id}/media [post]
func (h *MediaHandler) UploadTemplateMedia(w http.ResponseWriter, r *http.Request) {
	templateID, parseErr := uuid.Parse(r.PathValue("id"))
	if parseErr != nil {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidTemplateIDDetail())
		return
	}

	file, contentType, err := h.parseUpload(w, r)
	if err != nil {
		h.writeUploadParseError(w, err)
		return
	}
	defer file.Close()

	if !contentTypeAllowed(true, contentType) {
		handler.WriteError(w, h.logger, http.StatusBadRequest, unsupportedMediaTypeDetail())
		return
	}

	key := h.storage.GenerateKey("templates/" + templateID.String())
	err = h.storage.Upload(r.Context(), key, file, contentType)
	if err != nil {
		handler.WriteSystemError(w, h.logger, http.StatusInternalServerError, err)
		return
	}

	media, err := h.templateSvc.UploadMedia(r.Context(), servicetemplate.UploadTemplateMediaCommand{
		TemplateID: templateID,
		UserID:     handler.UserIDFromContext(r.Context()),
		S3Key:      key,
	})
	if err != nil {
		h.deleteUploadedObject(r.Context(), key)
		switch {
		case errors.Is(err, domaintemplate.ErrTemplateNotFound):
			handler.WriteError(w, h.logger, http.StatusNotFound, templateNotFoundDetail(err))
		case errors.Is(err, domaintemplate.ErrNotOwner):
			handler.WriteError(w, h.logger, http.StatusForbidden, forbiddenDetail(err))
		case errors.Is(err, domaintemplate.ErrInvalidS3Key):
			handler.WriteError(w, h.logger, http.StatusBadRequest, invalidRequestBodyDetail())
		default:
			handler.WriteSystemError(w, h.logger, http.StatusInternalServerError, err)
		}
		return
	}

	handler.WriteJSON(w, h.logger, http.StatusCreated, toTemplateMediaResponse(media))
}

// parseUpload parses the multipart request, enforcing the size limit, and
// returns the uploaded file stream and its declared content type.
func (h *MediaHandler) parseUpload(
	w http.ResponseWriter,
	r *http.Request,
) (io.ReadCloser, string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	//nolint:gosec // G120: the body is capped by http.MaxBytesReader before parsing.
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return nil, "", errUploadTooLarge
		}
		return nil, "", errMalformedMultipart
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return nil, "", errMissingFile
		}
		return nil, "", errMalformedMultipart
	}
	return file, header.Header.Get("Content-Type"), nil
}

// parseMediaType validates the declared media type form field.
func parseMediaType(raw string) (domainexercise.MediaType, error) {
	switch domainexercise.MediaType(raw) {
	case domainexercise.MediaTypePhoto, domainexercise.MediaTypeVideo:
		return domainexercise.MediaType(raw), nil
	default:
		return "", errInvalidMediaType
	}
}

// contentTypeAllowed reports whether the uploaded file's content type is
// allowed for the declared kind: photos are images, videos are mp4.
func contentTypeAllowed(photo bool, contentType string) bool {
	allowed := []string{"image/jpeg", "image/png", "image/webp"}
	if !photo {
		allowed = []string{"video/mp4"}
	}
	return slices.Contains(allowed, contentType)
}

// writeUploadParseError maps multipart parsing failures to HTTP responses.
func (h *MediaHandler) writeUploadParseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errUploadTooLarge):
		handler.WriteError(w, h.logger, http.StatusRequestEntityTooLarge, payloadTooLargeDetail())
	case errors.Is(err, errMissingFile):
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidRequestBodyDetail())
	default:
		handler.WriteError(w, h.logger, http.StatusBadRequest, malformedMultipartDetail())
	}
}

// deleteUploadedObject best-effort removes an object that was uploaded but
// never linked to a database record. The caller's original error is
// preserved; a failed delete is only logged.
func (h *MediaHandler) deleteUploadedObject(ctx context.Context, key string) {
	if err := h.storage.Delete(ctx, key); err != nil {
		h.logger.Warn("failed to delete orphaned media object", "s3_key", key, "error", err)
	}
}
