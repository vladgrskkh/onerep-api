package template

import "github.com/vladgrskkh/onerep-api/internal/handler"

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

	errCodeTemplateNotFound = "TEMPLATE_NOT_FOUND"
	errUserTemplateNotFound = "Template not found"

	errCodeForbidden = "FORBIDDEN"
	errUserForbidden = "You do not have access to this template"

	errCodeInvalidName = "INVALID_NAME"
	errUserInvalidName = "Name must not be empty"

	errCodeInvalidUserID = "INVALID_USER_ID"
	errUserInvalidUserID = "You must be logged in to create templates"
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

func templateNotFoundDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeTemplateNotFound,
		Message:     err.Error(),
		UserMessage: errUserTemplateNotFound,
	}
}

func forbiddenDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeForbidden,
		Message:     err.Error(),
		UserMessage: errUserForbidden,
	}
}

func invalidNameDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidName,
		Message:     err.Error(),
		UserMessage: errUserInvalidName,
	}
}

func invalidUserIDDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidUserID,
		Message:     err.Error(),
		UserMessage: errUserInvalidUserID,
	}
}
