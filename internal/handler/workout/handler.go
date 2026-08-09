package workout

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/vladgrskkh/onerep-api/internal/handler"
	"github.com/vladgrskkh/onerep-api/internal/handler/workout/dto"
)

type WorkoutHandler struct {
	svc    WorkoutService
	logger *slog.Logger
}

func NewWorkoutHandler(svc WorkoutService, logger *slog.Logger) *WorkoutHandler {
	return &WorkoutHandler{svc: svc, logger: logger}
}

// Start begins a new workout for the authenticated user, optionally copying
// the exercises of a template. An empty request body starts a workout
// without a template.
//
// @Summary Start a workout
// @Description Start a workout for the authenticated user, optionally from a template; an empty body starts without a template
// @Tags workouts
// @Accept json
// @Produce json
// @Param request body dto.StartWorkoutRequest false "Template ID, optional"
// @Success 201 {object} dto.WorkoutResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 404 {object} handler.ErrorResponse
// @Failure 409 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /workouts [post]
func (h *WorkoutHandler) Start(w http.ResponseWriter, r *http.Request) {
	var req dto.StartWorkoutRequest
	if r.ContentLength != 0 {
		if err := handler.DecodeAndValidate(r, &req); err != nil {
			if errors.Is(err, handler.ErrValidationFailed) {
				handler.WriteError(w, h.logger, http.StatusBadRequest, handler.ValidationErrorDetail(err))
				return
			}
			handler.WriteError(w, h.logger, http.StatusBadRequest, invalidRequestBodyDetail())
			return
		}
	}

	userID := handler.UserIDFromContext(r.Context())
	workout, err := h.svc.Start(r.Context(), toStartCommand(req, userID))
	if err != nil {
		status, detail := mapError(err)
		handler.WriteError(w, h.logger, status, detail)
		return
	}

	handler.WriteJSON(w, h.logger, http.StatusCreated, toWorkoutResponse(workout))
}

// List returns the authenticated user's workouts.
//
// @Summary List workouts
// @Description List the authenticated user's workouts, optionally filtered by last-updated time
// @Tags workouts
// @Accept json
// @Produce json
// @Param since query string false "Only workouts updated after this RFC 3339 timestamp"
// @Success 200 {array} dto.WorkoutResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /workouts [get]
func (h *WorkoutHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := handler.UserIDFromContext(r.Context())

	var since *time.Time
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		parsed, parseErr := time.Parse(time.RFC3339, sinceStr)
		if parseErr != nil {
			handler.WriteError(w, h.logger, http.StatusBadRequest, invalidSinceDetail())
			return
		}
		since = &parsed
	}

	workouts, err := h.svc.List(r.Context(), userID, since)
	if err != nil {
		status, detail := mapError(err)
		handler.WriteError(w, h.logger, status, detail)
		return
	}

	handler.WriteJSON(w, h.logger, http.StatusOK, toWorkoutListResponse(workouts))
}

// Get returns a single workout with its exercises and sets.
//
// @Summary Get a workout
// @Description Get a workout by ID, including its exercises and sets
// @Tags workouts
// @Accept json
// @Produce json
// @Param id path string true "Workout ID"
// @Success 200 {object} dto.WorkoutResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 404 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /workouts/{id} [get]
func (h *WorkoutHandler) Get(w http.ResponseWriter, r *http.Request) {
	workoutID, parseErr := uuid.Parse(r.PathValue("id"))
	if parseErr != nil {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidWorkoutIDDetail())
		return
	}

	userID := handler.UserIDFromContext(r.Context())
	workout, err := h.svc.Get(r.Context(), workoutID, userID)
	if err != nil {
		status, detail := mapError(err)
		handler.WriteError(w, h.logger, status, detail)
		return
	}

	handler.WriteJSON(w, h.logger, http.StatusOK, toWorkoutResponse(workout))
}

// AddExercise appends an exercise to the authenticated user's workout.
//
// @Summary Add an exercise to a workout
// @Description Append an exercise to the authenticated user's workout
// @Tags workouts
// @Accept json
// @Produce json
// @Param id path string true "Workout ID"
// @Param request body dto.AddExerciseRequest true "Exercise to add"
// @Success 201 {object} dto.WorkoutExerciseResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 404 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /workouts/{id}/exercises [post]
func (h *WorkoutHandler) AddExercise(w http.ResponseWriter, r *http.Request) {
	workoutID, parseErr := uuid.Parse(r.PathValue("id"))
	if parseErr != nil {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidWorkoutIDDetail())
		return
	}

	var req dto.AddExerciseRequest
	if err := handler.DecodeAndValidate(r, &req); err != nil {
		if errors.Is(err, handler.ErrValidationFailed) {
			handler.WriteError(w, h.logger, http.StatusBadRequest, handler.ValidationErrorDetail(err))
			return
		}
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidRequestBodyDetail())
		return
	}

	userID := handler.UserIDFromContext(r.Context())
	we, err := h.svc.AddExercise(r.Context(), toAddExerciseCommand(req, workoutID), userID)
	if err != nil {
		status, detail := mapError(err)
		handler.WriteError(w, h.logger, status, detail)
		return
	}

	handler.WriteJSON(w, h.logger, http.StatusCreated, toWorkoutExerciseResponse(we))
}

// LogSet records a set and returns its PR status with the estimated 1RM.
//
// @Summary Log a set
// @Description Log a set against a workout exercise; returns the estimated one-rep max (Epley formula) and whether it is a new personal record
// @Tags workouts
// @Accept json
// @Produce json
// @Param id path string true "Workout ID"
// @Param exId path string true "Workout exercise ID"
// @Param request body dto.LogSetRequest true "Set data"
// @Success 201 {object} dto.LogSetResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 404 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /workouts/{id}/exercises/{exId}/sets [post]
func (h *WorkoutHandler) LogSet(w http.ResponseWriter, r *http.Request) {
	workoutID, parseErr := uuid.Parse(r.PathValue("id"))
	if parseErr != nil {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidWorkoutIDDetail())
		return
	}
	workoutExerciseID, parseErr := uuid.Parse(r.PathValue("exId"))
	if parseErr != nil {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidWorkoutExerciseIDDetail())
		return
	}

	var req dto.LogSetRequest
	if err := handler.DecodeAndValidate(r, &req); err != nil {
		if errors.Is(err, handler.ErrValidationFailed) {
			handler.WriteError(w, h.logger, http.StatusBadRequest, handler.ValidationErrorDetail(err))
			return
		}
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidRequestBodyDetail())
		return
	}

	userID := handler.UserIDFromContext(r.Context())
	result, err := h.svc.LogSet(r.Context(), toLogSetCommand(req, workoutID, workoutExerciseID), userID)
	if err != nil {
		status, detail := mapError(err)
		handler.WriteError(w, h.logger, status, detail)
		return
	}

	handler.WriteJSON(w, h.logger, http.StatusCreated, toLogSetResponse(result))
}

// Finish marks the authenticated user's workout as completed.
//
// @Summary Finish a workout
// @Description Mark the authenticated user's workout as completed and enqueue its volume recalculation
// @Tags workouts
// @Accept json
// @Produce json
// @Param id path string true "Workout ID"
// @Success 200 {object} dto.WorkoutResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 404 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /workouts/{id}/finish [patch]
func (h *WorkoutHandler) Finish(w http.ResponseWriter, r *http.Request) {
	workoutID, parseErr := uuid.Parse(r.PathValue("id"))
	if parseErr != nil {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidWorkoutIDDetail())
		return
	}

	userID := handler.UserIDFromContext(r.Context())
	workout, err := h.svc.Finish(r.Context(), toFinishCommand(workoutID), userID)
	if err != nil {
		status, detail := mapError(err)
		handler.WriteError(w, h.logger, status, detail)
		return
	}

	handler.WriteJSON(w, h.logger, http.StatusOK, toWorkoutResponse(workout))
}
