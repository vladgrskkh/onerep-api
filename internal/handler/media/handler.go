package media

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	domainexercise "github.com/vladgrskkh/onerep-api/internal/domain/exercise"
	domaintemplate "github.com/vladgrskkh/onerep-api/internal/domain/template"
	"github.com/vladgrskkh/onerep-api/internal/handler"
	"github.com/vladgrskkh/onerep-api/internal/handler/media/dto"
	serviceexercise "github.com/vladgrskkh/onerep-api/internal/service/exercise"
	servicetemplate "github.com/vladgrskkh/onerep-api/internal/service/template"
)

// ExerciseService is the exercise media upload use case consumed by the
// media handler.
type ExerciseService interface {
	UploadMedia(
		ctx context.Context,
		cmd serviceexercise.UploadExerciseMediaCommand,
	) (*serviceexercise.ExerciseMediaUpload, error)
}

// TemplateService is the template media upload use case consumed by the
// media handler.
type TemplateService interface {
	UploadMedia(
		ctx context.Context,
		cmd servicetemplate.UploadTemplateMediaCommand,
	) (*servicetemplate.TemplateMediaUpload, error)
}

// MediaHandler registers media objects for exercises and templates and
// returns the presigned URLs the client uploads the files to.
type MediaHandler struct {
	exerciseSvc ExerciseService
	templateSvc TemplateService
	logger      *slog.Logger
}

func NewMediaHandler(
	exerciseSvc ExerciseService,
	templateSvc TemplateService,
	logger *slog.Logger,
) *MediaHandler {
	return &MediaHandler{
		exerciseSvc: exerciseSvc,
		templateSvc: templateSvc,
		logger:      logger,
	}
}

// UploadExerciseMedia registers media for an exercise and returns the
// presigned URL the client uploads the file to.
//
// @Summary Create exercise media upload
// @Description Register a photo or video for an exercise and return a presigned URL for uploading the file to the media bucket
// @Tags exercises
// @Accept json
// @Produce json
// @Param id path string true "Exercise ID"
// @Param request body dto.ExerciseMediaUploadRequest true "Media upload data"
// @Success 201 {object} dto.MediaUploadResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 404 {object} handler.ErrorResponse
// @Failure 401 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /exercises/{id}/media [post]
func (h *MediaHandler) UploadExerciseMedia(w http.ResponseWriter, r *http.Request) {
	exerciseID, parseErr := uuid.Parse(r.PathValue("id"))
	if parseErr != nil {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidExerciseIDDetail())
		return
	}

	var req dto.ExerciseMediaUploadRequest
	if err := handler.DecodeAndValidate(r, &req); err != nil {
		if errors.Is(err, handler.ErrValidationFailed) {
			handler.WriteError(
				w,
				h.logger,
				http.StatusBadRequest,
				handler.ValidationErrorDetail(err),
			)
			return
		}
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidRequestBodyDetail())
		return
	}

	upload, err := h.exerciseSvc.UploadMedia(
		r.Context(),
		serviceexercise.UploadExerciseMediaCommand{
			ExerciseID:  exerciseID,
			MediaType:   domainexercise.MediaType(req.MediaType),
			ContentType: req.ContentType,
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, domainexercise.ErrExerciseNotFound):
			handler.WriteError(w, h.logger, http.StatusNotFound, exerciseNotFoundDetail(err))
		case errors.Is(err, domainexercise.ErrCannotEditBuiltIn):
			handler.WriteError(w, h.logger, http.StatusBadRequest, cannotEditBuiltInDetail(err))
		case errors.Is(err, domainexercise.ErrInvalidMediaType):
			handler.WriteError(w, h.logger, http.StatusBadRequest, invalidMediaTypeDetail(err))
		case errors.Is(err, domainexercise.ErrUnsupportedContentType):
			handler.WriteError(w, h.logger, http.StatusBadRequest, unsupportedMediaTypeDetail())
		default:
			handler.WriteSystemError(w, h.logger, http.StatusInternalServerError, err)
		}
		return
	}

	handler.WriteJSON(
		w,
		h.logger,
		http.StatusCreated,
		toExerciseMediaUploadResponse(upload.Media, upload.UploadURL, upload.ExpiresIn),
	)
}

// UploadTemplateMedia registers media for a template and returns the
// presigned URL the client uploads the file to.
//
// @Summary Create template media upload
// @Description Register a photo for a template and return a presigned URL for uploading the file to the media bucket
// @Tags templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Param request body dto.TemplateMediaUploadRequest true "Media upload data"
// @Success 201 {object} dto.MediaUploadResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 403 {object} handler.ErrorResponse
// @Failure 404 {object} handler.ErrorResponse
// @Failure 401 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /templates/{id}/media [post]
func (h *MediaHandler) UploadTemplateMedia(w http.ResponseWriter, r *http.Request) {
	templateID, parseErr := uuid.Parse(r.PathValue("id"))
	if parseErr != nil {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidTemplateIDDetail())
		return
	}

	var req dto.TemplateMediaUploadRequest
	if err := handler.DecodeAndValidate(r, &req); err != nil {
		if errors.Is(err, handler.ErrValidationFailed) {
			handler.WriteError(
				w,
				h.logger,
				http.StatusBadRequest,
				handler.ValidationErrorDetail(err),
			)
			return
		}
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidRequestBodyDetail())
		return
	}

	upload, err := h.templateSvc.UploadMedia(
		r.Context(),
		servicetemplate.UploadTemplateMediaCommand{
			TemplateID:  templateID,
			UserID:      handler.UserIDFromContext(r.Context()),
			ContentType: req.ContentType,
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, domaintemplate.ErrTemplateNotFound):
			handler.WriteError(w, h.logger, http.StatusNotFound, templateNotFoundDetail(err))
		case errors.Is(err, domaintemplate.ErrNotOwner):
			handler.WriteError(w, h.logger, http.StatusForbidden, forbiddenDetail(err))
		case errors.Is(err, domaintemplate.ErrUnsupportedContentType):
			handler.WriteError(w, h.logger, http.StatusBadRequest, unsupportedMediaTypeDetail())
		default:
			handler.WriteSystemError(w, h.logger, http.StatusInternalServerError, err)
		}
		return
	}

	handler.WriteJSON(
		w,
		h.logger,
		http.StatusCreated,
		toTemplateMediaUploadResponse(upload.Media, upload.UploadURL, upload.ExpiresIn),
	)
}
