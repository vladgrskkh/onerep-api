package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
)

type ErrorCode string

type ErrorDetail struct {
	Code        ErrorCode          `json:"code"`
	Message     string             `json:"message"`
	UserMessage string             `json:"user_message,omitempty"`
	Details     []ValidationDetail `json:"details,omitempty"`
}

// ValidationDetail describes a single failed request field validation.
type ValidationDetail struct {
	Field string `json:"field,omitempty"`
	Tag   string `json:"tag,omitempty"`
	Param string `json:"param,omitempty"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// WriteJSON writes a JSON response. Once WriteHeader is called the response is
// committed; an encoding failure at this point is not recoverable server-side,
// so we only log it.
func WriteJSON(w http.ResponseWriter, logger *slog.Logger, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logger.Error("failed to encode response", "error", err)
	}
}

func WriteError(w http.ResponseWriter, logger *slog.Logger, status int, detail ErrorDetail) {
	WriteJSON(w, logger, status, ErrorResponse{Error: detail})
}

func WriteSystemError(w http.ResponseWriter, logger *slog.Logger, status int, err error) {
	WriteError(w, logger, status, ErrorDetail{
		Code:    "INTERNAL_ERROR",
		Message: err.Error(),
	})
}

// ValidationErrorDetail builds the 400 response for a failed request
// validation, carrying per-field details extracted from the validator error.
func ValidationErrorDetail(err error) ErrorDetail {
	detail := ErrorDetail{
		Code:        "VALIDATION_ERROR",
		Message:     "request validation failed",
		UserMessage: "Please check your input",
	}

	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		for _, fe := range verrs {
			detail.Details = append(detail.Details, ValidationDetail{
				Field: fe.Field(),
				Tag:   fe.Tag(),
				Param: fe.Param(),
			})
		}
	}

	return detail
}
