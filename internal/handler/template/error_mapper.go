package template

import (
	"errors"
	"net/http"

	domaintemplate "github.com/vladgrskkh/onerep-api/internal/domain/template"
	"github.com/vladgrskkh/onerep-api/internal/handler"
)

const (
	errCodeInvalidRequestBody = "INVALID_REQUEST_BODY"
	errMsgInvalidRequestBody  = "invalid request body"
	errUserInvalidRequestBody = "The request body is invalid"

	errCodeInvalidTemplateID = "INVALID_TEMPLATE_ID"
	errMsgInvalidTemplateID  = "invalid template id"
	errUserInvalidTemplateID = "The template ID is invalid"

	errCodeInvalidSince = "INVALID_SINCE"
	errMsgInvalidSince  = "invalid since parameter"
	errUserInvalidSince = "The since parameter must be an RFC 3339 timestamp"
)

func invalidRequestBodyDetail() handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidRequestBody,
		Message:     errMsgInvalidRequestBody,
		UserMessage: errUserInvalidRequestBody,
	}
}

func invalidTemplateIDDetail() handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidTemplateID,
		Message:     errMsgInvalidTemplateID,
		UserMessage: errUserInvalidTemplateID,
	}
}

func invalidSinceDetail() handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidSince,
		Message:     errMsgInvalidSince,
		UserMessage: errUserInvalidSince,
	}
}

func mapError(err error) (int, handler.ErrorDetail) {
	switch {
	case errors.Is(err, domaintemplate.ErrTemplateNotFound):
		return http.StatusNotFound, handler.ErrorDetail{
			Code:        "TEMPLATE_NOT_FOUND",
			Message:     err.Error(),
			UserMessage: "Template not found",
		}
	case errors.Is(err, domaintemplate.ErrNotOwner):
		return http.StatusForbidden, handler.ErrorDetail{
			Code:        "FORBIDDEN",
			Message:     err.Error(),
			UserMessage: "You do not have access to this template",
		}
	case errors.Is(err, domaintemplate.ErrInvalidName):
		return http.StatusBadRequest, handler.ErrorDetail{
			Code:        "INVALID_NAME",
			Message:     err.Error(),
			UserMessage: "Name must not be empty",
		}
	case errors.Is(err, domaintemplate.ErrInvalidUserID):
		return http.StatusBadRequest, handler.ErrorDetail{
			Code:        "INVALID_USER_ID",
			Message:     err.Error(),
			UserMessage: "You must be logged in to create templates",
		}
	default:
		return http.StatusInternalServerError, handler.ErrorDetail{
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
		}
	}
}
