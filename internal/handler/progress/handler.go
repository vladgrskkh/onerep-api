package progress

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	domainbodyweight "github.com/vladgrskkh/onerep-api/internal/domain/bodyweight"
	domainprogress "github.com/vladgrskkh/onerep-api/internal/domain/progress"
	"github.com/vladgrskkh/onerep-api/internal/handler"
	"github.com/vladgrskkh/onerep-api/internal/handler/progress/dto"
	servicebodyweight "github.com/vladgrskkh/onerep-api/internal/service/bodyweight"
)

// ProgressService is the progress read-model contract consumed by the
// handler.
type ProgressService interface {
	Get1RM(
		ctx context.Context,
		userID, exerciseID uuid.UUID,
		from, to *time.Time,
	) ([]*domainprogress.Progress1RM, error)
	GetVolume(ctx context.Context, userID uuid.UUID, from, to *time.Time) ([]*domainprogress.ProgressVolume, error)
}

// BodyWeightService is the body weight contract consumed by the handler.
type BodyWeightService interface {
	LogBodyWeight(ctx context.Context, cmd servicebodyweight.LogBodyWeightCommand) (*domainbodyweight.BodyWeight, error)
	ListBodyWeight(ctx context.Context, userID uuid.UUID, since *time.Time) ([]*domainbodyweight.BodyWeight, error)
}

type ProgressHandler struct {
	progress   ProgressService
	bodyWeight BodyWeightService
	logger     *slog.Logger
}

func NewProgressHandler(
	progress ProgressService,
	bodyWeight BodyWeightService,
	logger *slog.Logger,
) *ProgressHandler {
	return &ProgressHandler{progress: progress, bodyWeight: bodyWeight, logger: logger}
}

// Get1RM returns the user's estimated one-rep max history for an exercise.
//
// @Summary Get 1RM progress
// @Description Get the authenticated user's estimated one-rep max history for an exercise, optionally filtered by date range
// @Tags progress
// @Accept json
// @Produce json
// @Param exercise_id query string true "Exercise ID"
// @Param from query string false "Start of the date range (RFC 3339)"
// @Param to query string false "End of the date range (RFC 3339)"
// @Success 200 {array} dto.OneRMResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /progress/1rm [get]
func (h *ProgressHandler) Get1RM(w http.ResponseWriter, r *http.Request) {
	exerciseID, parseErr := uuid.Parse(r.URL.Query().Get("exercise_id"))
	if parseErr != nil {
		handler.WriteError(
			w,
			h.logger,
			http.StatusBadRequest,
			invalidExerciseIDDetail(errors.New(errMsgInvalidExerciseID)),
		)
		return
	}

	fromVal, fromPresent, parseErr := handler.ParseRFC3339QueryParam(r, "from")
	if parseErr != nil {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidDateRangeDetail())
		return
	}
	var from *time.Time
	if fromPresent {
		from = &fromVal
	}
	toVal, toPresent, parseErr := handler.ParseRFC3339QueryParam(r, "to")
	if parseErr != nil {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidDateRangeDetail())
		return
	}
	var to *time.Time
	if toPresent {
		to = &toVal
	}
	if from != nil && to != nil && from.After(*to) {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidDateRangeDetail())
		return
	}

	userID := handler.UserIDFromContext(r.Context())
	progress, err := h.progress.Get1RM(r.Context(), userID, exerciseID, from, to)
	if err != nil {
		switch {
		case errors.Is(err, domainprogress.ErrInvalidExerciseID):
			handler.WriteError(w, h.logger, http.StatusBadRequest, invalidExerciseIDDetail(err))
		case errors.Is(err, domainprogress.ErrInvalidUserID):
			handler.WriteError(w, h.logger, http.StatusBadRequest, invalidUserIDDetail(err))
		default:
			handler.WriteSystemError(w, h.logger, http.StatusInternalServerError, err)
		}
		return
	}

	handler.WriteJSON(w, h.logger, http.StatusOK, toOneRMListResponse(progress))
}

// GetVolume returns the user's total volume per muscle group.
//
// @Summary Get volume progress
// @Description Get the authenticated user's total volume per muscle group, optionally filtered by date range
// @Tags progress
// @Accept json
// @Produce json
// @Param from query string false "Start of the date range (RFC 3339)"
// @Param to query string false "End of the date range (RFC 3339)"
// @Success 200 {array} dto.VolumeResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /progress/volume [get]
func (h *ProgressHandler) GetVolume(w http.ResponseWriter, r *http.Request) {
	fromVal, fromPresent, parseErr := handler.ParseRFC3339QueryParam(r, "from")
	if parseErr != nil {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidDateRangeDetail())
		return
	}
	var from *time.Time
	if fromPresent {
		from = &fromVal
	}
	toVal, toPresent, parseErr := handler.ParseRFC3339QueryParam(r, "to")
	if parseErr != nil {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidDateRangeDetail())
		return
	}
	var to *time.Time
	if toPresent {
		to = &toVal
	}
	if from != nil && to != nil && from.After(*to) {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidDateRangeDetail())
		return
	}

	userID := handler.UserIDFromContext(r.Context())
	volumes, err := h.progress.GetVolume(r.Context(), userID, from, to)
	if err != nil {
		switch {
		case errors.Is(err, domainprogress.ErrInvalidUserID):
			handler.WriteError(w, h.logger, http.StatusBadRequest, invalidUserIDDetail(err))
		default:
			handler.WriteSystemError(w, h.logger, http.StatusInternalServerError, err)
		}
		return
	}

	handler.WriteJSON(w, h.logger, http.StatusOK, toVolumeListResponse(volumes))
}

// GetBodyWeight returns the user's body weight entries.
//
// @Summary List body weight entries
// @Description List the authenticated user's body weight entries, optionally filtered by last-updated time
// @Tags progress
// @Accept json
// @Produce json
// @Param since query string false "Only entries updated after this RFC 3339 timestamp"
// @Success 200 {array} dto.BodyWeightResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /progress/body-weight [get]
func (h *ProgressHandler) GetBodyWeight(w http.ResponseWriter, r *http.Request) {
	sinceVal, present, parseErr := handler.ParseRFC3339QueryParam(r, "since")
	if parseErr != nil {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidSinceDetail())
		return
	}
	var since *time.Time
	if present {
		since = &sinceVal
	}

	userID := handler.UserIDFromContext(r.Context())
	weights, err := h.bodyWeight.ListBodyWeight(r.Context(), userID, since)
	if err != nil {
		switch {
		case errors.Is(err, domainbodyweight.ErrInvalidUserID):
			handler.WriteError(w, h.logger, http.StatusBadRequest, invalidUserIDDetail(err))
		default:
			handler.WriteSystemError(w, h.logger, http.StatusInternalServerError, err)
		}
		return
	}

	handler.WriteJSON(w, h.logger, http.StatusOK, toBodyWeightListResponse(weights))
}

// LogBodyWeight records a body weight entry.
//
// @Summary Log a body weight
// @Description Record a body weight entry for the authenticated user; measured_at defaults to now
// @Tags progress
// @Accept json
// @Produce json
// @Param request body dto.LogBodyWeightRequest true "Body weight data"
// @Success 201 {object} dto.BodyWeightResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /progress/body-weight [post]
func (h *ProgressHandler) LogBodyWeight(w http.ResponseWriter, r *http.Request) {
	var req dto.LogBodyWeightRequest
	if err := handler.DecodeAndValidate(r, &req); err != nil {
		if errors.Is(err, handler.ErrValidationFailed) {
			handler.WriteError(w, h.logger, http.StatusBadRequest, handler.ValidationErrorDetail(err))
			return
		}
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidRequestBodyDetail())
		return
	}

	userID := handler.UserIDFromContext(r.Context())
	bw, err := h.bodyWeight.LogBodyWeight(r.Context(), toLogBodyWeightCommand(req, userID))
	if err != nil {
		switch {
		case errors.Is(err, domainbodyweight.ErrInvalidUserID):
			handler.WriteError(w, h.logger, http.StatusBadRequest, invalidUserIDDetail(err))
		case errors.Is(err, domainbodyweight.ErrInvalidWeight):
			handler.WriteError(w, h.logger, http.StatusBadRequest, invalidWeightDetail(err))
		default:
			handler.WriteSystemError(w, h.logger, http.StatusInternalServerError, err)
		}
		return
	}

	handler.WriteJSON(w, h.logger, http.StatusCreated, toBodyWeightResponse(bw))
}
