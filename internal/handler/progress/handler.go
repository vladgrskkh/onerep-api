package progress

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/vladgrskkh/onerep-api/internal/handler"
	"github.com/vladgrskkh/onerep-api/internal/handler/progress/dto"
)

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

// parseTimeParam parses an RFC 3339 query parameter.
func parseTimeParam(raw string) (*time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
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
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidExerciseIDDetail())
		return
	}

	var from *time.Time
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		from, parseErr = parseTimeParam(fromStr)
		if parseErr != nil {
			handler.WriteError(w, h.logger, http.StatusBadRequest, invalidDateRangeDetail())
			return
		}
	}
	var to *time.Time
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		to, parseErr = parseTimeParam(toStr)
		if parseErr != nil {
			handler.WriteError(w, h.logger, http.StatusBadRequest, invalidDateRangeDetail())
			return
		}
	}

	userID := handler.UserIDFromContext(r.Context())
	progress, err := h.progress.Get1RM(r.Context(), userID, exerciseID, from, to)
	if err != nil {
		status, detail := mapError(err)
		handler.WriteError(w, h.logger, status, detail)
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
	var parseErr error
	var from *time.Time
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		from, parseErr = parseTimeParam(fromStr)
		if parseErr != nil {
			handler.WriteError(w, h.logger, http.StatusBadRequest, invalidDateRangeDetail())
			return
		}
	}
	var to *time.Time
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		to, parseErr = parseTimeParam(toStr)
		if parseErr != nil {
			handler.WriteError(w, h.logger, http.StatusBadRequest, invalidDateRangeDetail())
			return
		}
	}

	userID := handler.UserIDFromContext(r.Context())
	volumes, err := h.progress.GetVolume(r.Context(), userID, from, to)
	if err != nil {
		status, detail := mapError(err)
		handler.WriteError(w, h.logger, status, detail)
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
	var since *time.Time
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		var parseErr error
		since, parseErr = parseTimeParam(sinceStr)
		if parseErr != nil {
			handler.WriteError(w, h.logger, http.StatusBadRequest, invalidSinceDetail())
			return
		}
	}

	userID := handler.UserIDFromContext(r.Context())
	weights, err := h.bodyWeight.ListBodyWeight(r.Context(), userID, since)
	if err != nil {
		status, detail := mapError(err)
		handler.WriteError(w, h.logger, status, detail)
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
		status, detail := mapError(err)
		handler.WriteError(w, h.logger, status, detail)
		return
	}

	handler.WriteJSON(w, h.logger, http.StatusCreated, toBodyWeightResponse(bw))
}
