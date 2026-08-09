package template

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	domaintemplate "github.com/vladgrskkh/onerep-api/internal/domain/template"
	"github.com/vladgrskkh/onerep-api/internal/handler"
	"github.com/vladgrskkh/onerep-api/internal/handler/template/dto"
)

type TemplateHandler struct {
	svc    TemplateService
	logger *slog.Logger
}

func NewTemplateHandler(svc TemplateService, logger *slog.Logger) *TemplateHandler {
	return &TemplateHandler{svc: svc, logger: logger}
}

// List returns templates matching the query filters.
//
// @Summary List templates
// @Description List the authenticated user's templates, optionally filtered by last-updated time
// @Tags templates
// @Accept json
// @Produce json
// @Param since query string false "Only templates updated after this RFC 3339 timestamp"
// @Success 200 {array} dto.TemplateResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /templates [get]
func (h *TemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	var filter domaintemplate.TemplateFilter
	if userID := handler.UserIDFromContext(r.Context()); userID != uuid.Nil {
		filter.UserID = &userID
	}
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		since, parseErr := time.Parse(time.RFC3339, sinceStr)
		if parseErr != nil {
			handler.WriteError(w, h.logger, http.StatusBadRequest, invalidSinceDetail())
			return
		}
		filter.Since = &since
	}

	templates, err := h.svc.List(r.Context(), filter)
	if err != nil {
		status, detail := mapError(err)
		handler.WriteError(w, h.logger, status, detail)
		return
	}

	handler.WriteJSON(w, h.logger, http.StatusOK, toTemplateListResponse(templates))
}

// Get returns a single template.
//
// @Summary Get a template
// @Description Get a template by ID, including its exercises and media
// @Tags templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Success 200 {object} dto.TemplateResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 404 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /templates/{id} [get]
func (h *TemplateHandler) Get(w http.ResponseWriter, r *http.Request) {
	templateID, parseErr := uuid.Parse(r.PathValue("id"))
	if parseErr != nil {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidTemplateIDDetail())
		return
	}

	t, err := h.svc.Get(r.Context(), templateID)
	if err != nil {
		status, detail := mapError(err)
		handler.WriteError(w, h.logger, status, detail)
		return
	}

	handler.WriteJSON(w, h.logger, http.StatusOK, toTemplateResponse(t))
}

// Create creates a template for the authenticated user.
//
// @Summary Create a template
// @Description Create a custom template with a list of exercises and planned sets
// @Tags templates
// @Accept json
// @Produce json
// @Param request body dto.TemplateCreateRequest true "Template data"
// @Success 201 {object} dto.TemplateResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /templates [post]
func (h *TemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.TemplateCreateRequest
	if err := handler.DecodeAndValidate(r, &req); err != nil {
		if errors.Is(err, handler.ErrValidationFailed) {
			handler.WriteError(w, h.logger, http.StatusBadRequest, handler.ValidationErrorDetail(err))
			return
		}
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidRequestBodyDetail())
		return
	}

	userID := handler.UserIDFromContext(r.Context())
	t, err := h.svc.Create(r.Context(), toCreateCommand(req, userID))
	if err != nil {
		status, detail := mapError(err)
		handler.WriteError(w, h.logger, status, detail)
		return
	}

	handler.WriteJSON(w, h.logger, http.StatusCreated, toTemplateResponse(t))
}

// Update patches a template.
//
// @Summary Update a template
// @Description Patch an existing template; null fields are left unchanged
// @Tags templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Param request body dto.TemplateUpdateRequest true "Fields to update"
// @Success 200 {object} dto.TemplateResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 403 {object} handler.ErrorResponse
// @Failure 404 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /templates/{id} [patch]
func (h *TemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	templateID, parseErr := uuid.Parse(r.PathValue("id"))
	if parseErr != nil {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidTemplateIDDetail())
		return
	}

	var req dto.TemplateUpdateRequest
	if err := handler.DecodeAndValidate(r, &req); err != nil {
		if errors.Is(err, handler.ErrValidationFailed) {
			handler.WriteError(w, h.logger, http.StatusBadRequest, handler.ValidationErrorDetail(err))
			return
		}
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidRequestBodyDetail())
		return
	}

	userID := handler.UserIDFromContext(r.Context())
	t, err := h.svc.Update(r.Context(), toUpdateCommand(req, templateID, userID))
	if err != nil {
		status, detail := mapError(err)
		handler.WriteError(w, h.logger, status, detail)
		return
	}

	handler.WriteJSON(w, h.logger, http.StatusOK, toTemplateResponse(t))
}

// Publish makes a template publicly visible.
//
// @Summary Publish a template
// @Description Make an owned template publicly visible to other users
// @Tags templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Success 200 {object} dto.TemplateResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 403 {object} handler.ErrorResponse
// @Failure 404 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /templates/{id}/publish [post]
func (h *TemplateHandler) Publish(w http.ResponseWriter, r *http.Request) {
	templateID, parseErr := uuid.Parse(r.PathValue("id"))
	if parseErr != nil {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidTemplateIDDetail())
		return
	}

	userID := handler.UserIDFromContext(r.Context())
	t, err := h.svc.Publish(r.Context(), templateID, userID)
	if err != nil {
		status, detail := mapError(err)
		handler.WriteError(w, h.logger, status, detail)
		return
	}

	handler.WriteJSON(w, h.logger, http.StatusOK, toTemplateResponse(t))
}

// Fork copies a template under the authenticated user.
//
// @Summary Fork a template
// @Description Copy a template, including its exercises and media, as the authenticated user's own
// @Tags templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Success 201 {object} dto.TemplateResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 404 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /templates/{id}/fork [post]
func (h *TemplateHandler) Fork(w http.ResponseWriter, r *http.Request) {
	templateID, parseErr := uuid.Parse(r.PathValue("id"))
	if parseErr != nil {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidTemplateIDDetail())
		return
	}

	userID := handler.UserIDFromContext(r.Context())
	t, err := h.svc.Fork(r.Context(), templateID, userID)
	if err != nil {
		status, detail := mapError(err)
		handler.WriteError(w, h.logger, status, detail)
		return
	}

	handler.WriteJSON(w, h.logger, http.StatusCreated, toTemplateResponse(t))
}

// SoftDelete marks a template as deleted.
//
// @Summary Delete a template
// @Description Soft-delete an owned template
// @Tags templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Success 204
// @Failure 400 {object} handler.ErrorResponse
// @Failure 403 {object} handler.ErrorResponse
// @Failure 404 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Security BearerAuth
// @Router /templates/{id} [delete]
func (h *TemplateHandler) SoftDelete(w http.ResponseWriter, r *http.Request) {
	templateID, parseErr := uuid.Parse(r.PathValue("id"))
	if parseErr != nil {
		handler.WriteError(w, h.logger, http.StatusBadRequest, invalidTemplateIDDetail())
		return
	}

	userID := handler.UserIDFromContext(r.Context())
	if err := h.svc.SoftDelete(r.Context(), templateID, userID); err != nil {
		status, detail := mapError(err)
		handler.WriteError(w, h.logger, status, detail)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
