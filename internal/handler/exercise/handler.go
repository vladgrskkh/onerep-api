package exercise

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	domainexercise "github.com/vladgrskkh/onerep-api/internal/domain/exercise"
	"github.com/vladgrskkh/onerep-api/internal/handler"
	"github.com/vladgrskkh/onerep-api/internal/handler/exercise/dto"
	serviceexercise "github.com/vladgrskkh/onerep-api/internal/service/exercise"
)

// ExerciseService is the exercise use-case contract consumed by the handler.
type ExerciseService interface {
	List(ctx context.Context, cmd serviceexercise.ListExercisesCommand) ([]domainexercise.Exercise, error)
	Get(ctx context.Context, id uuid.UUID) (domainexercise.Exercise, error)
	Create(ctx context.Context, cmd serviceexercise.CreateExerciseCommand) (domainexercise.Exercise, error)
	Update(ctx context.Context, cmd serviceexercise.UpdateExerciseCommand) (domainexercise.Exercise, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

type ExerciseHandler struct {
	svc    ExerciseService
	logger *slog.Logger
}

func NewExerciseHandler(svc ExerciseService, logger *slog.Logger) *ExerciseHandler {
	return &ExerciseHandler{svc: svc, logger: logger}
}

// List returns exercises matching the query filters.
//
// @Summary List exercises
// @Description List exercises, optionally filtered by search, muscle group, and last-updated time
// @Tags exercises
// @Accept json
// @Produce json
// @Param search query string false "Search by name"
// @Param muscle_group query string false "Filter by muscle group name"
// @Param since query string false "Only exercises updated after this RFC 3339 timestamp"
// @Success 200 {array} dto.ExerciseResponse
// @Failure 400 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /exercises [get]
func (h *ExerciseHandler) List(w http.ResponseWriter, r *http.Request) {
	cmd := serviceexercise.ListExercisesCommand{
		Search:      r.URL.Query().Get("search"),
		MuscleGroup: r.URL.Query().Get("muscle_group"),
	}

	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		since, parseErr := time.Parse(time.RFC3339, sinceStr)
		if parseErr != nil {
			handler.WriteError(w, h.logger, http.StatusBadRequest, invalidSinceDetail())
			return
		}
		cmd.Since = &since
	}

	exercises, err := h.svc.List(r.Context(), cmd)
	if err != nil {
		status, detail := mapError(err)
		handler.WriteError(w, h.logger, status, detail)
		return
	}

	handler.WriteJSON(w, h.logger, http.StatusOK, toExerciseListResponse(exercises))
}

// Get returns a single exercise.
//
// @Summary Get an exercise
// @Description Get an exercise by ID, including its media and muscle groups
// @Tags exercises
// @Accept json
// @Produce json
// @Param id path string true "Exercise ID"
// @Success 200 {object} dto.ExerciseResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 404 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /exercises/{id} [get]
func (h *ExerciseHandler) Get(w http.ResponseWriter, r *http.Request) {
	exerciseID, parseErr := uuid.Parse(r.PathValue("id"))
	if parseErr != nil {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidExerciseIDDetail())
		return
	}

	ex, err := h.svc.Get(r.Context(), exerciseID)
	if err != nil {
		status, detail := mapError(err)
		handler.WriteError(w, h.logger, status, detail)
		return
	}

	handler.WriteJSON(w, h.logger, http.StatusOK, toExerciseResponse(ex))
}

// Create creates a user exercise.
//
// @Summary Create an exercise
// @Description Create a custom exercise for the authenticated user
// @Tags exercises
// @Accept json
// @Produce json
// @Param request body dto.ExerciseCreateRequest true "Exercise data"
// @Success 201 {object} dto.ExerciseResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 404 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /exercises [post]
func (h *ExerciseHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.ExerciseCreateRequest
	if err := handler.DecodeAndValidate(r, &req); err != nil {
		if errors.Is(err, handler.ErrValidationFailed) {
			handler.WriteError(w, h.logger, http.StatusBadRequest, handler.ValidationErrorDetail(err))
			return
		}
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidRequestBodyDetail())
		return
	}

	userID := handler.UserIDFromContext(r.Context())
	ex, err := h.svc.Create(r.Context(), toCreateCommand(req, userID))
	if err != nil {
		status, detail := mapError(err)
		handler.WriteError(w, h.logger, status, detail)
		return
	}

	handler.WriteJSON(w, h.logger, http.StatusCreated, toExerciseResponse(ex))
}

// Update patches an exercise.
//
// @Summary Update an exercise
// @Description Patch an existing exercise; null fields are left unchanged
// @Tags exercises
// @Accept json
// @Produce json
// @Param id path string true "Exercise ID"
// @Param request body dto.ExerciseUpdateRequest true "Fields to update"
// @Success 200 {object} dto.ExerciseResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 404 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /exercises/{id} [patch]
func (h *ExerciseHandler) Update(w http.ResponseWriter, r *http.Request) {
	exerciseID, parseErr := uuid.Parse(r.PathValue("id"))
	if parseErr != nil {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidExerciseIDDetail())
		return
	}

	var req dto.ExerciseUpdateRequest
	if err := handler.DecodeAndValidate(r, &req); err != nil {
		if errors.Is(err, handler.ErrValidationFailed) {
			handler.WriteError(w, h.logger, http.StatusBadRequest, handler.ValidationErrorDetail(err))
			return
		}
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidRequestBodyDetail())
		return
	}

	ex, err := h.svc.Update(r.Context(), toUpdateCommand(req, exerciseID))
	if err != nil {
		status, detail := mapError(err)
		handler.WriteError(w, h.logger, status, detail)
		return
	}

	handler.WriteJSON(w, h.logger, http.StatusOK, toExerciseResponse(ex))
}

// SoftDelete marks an exercise as deleted.
//
// @Summary Delete an exercise
// @Description Soft-delete an existing exercise
// @Tags exercises
// @Accept json
// @Produce json
// @Param id path string true "Exercise ID"
// @Success 204
// @Failure 400 {object} handler.ErrorResponse
// @Failure 404 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /exercises/{id} [delete]
func (h *ExerciseHandler) SoftDelete(w http.ResponseWriter, r *http.Request) {
	exerciseID, parseErr := uuid.Parse(r.PathValue("id"))
	if parseErr != nil {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidExerciseIDDetail())
		return
	}

	if err := h.svc.SoftDelete(r.Context(), exerciseID); err != nil {
		status, detail := mapError(err)
		handler.WriteError(w, h.logger, status, detail)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
