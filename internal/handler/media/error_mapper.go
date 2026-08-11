package media

import (
	"errors"

	"github.com/vladgrskkh/onerep-api/internal/handler"
)

var (
	errUploadTooLarge     = errors.New("media upload exceeds the size limit")
	errMalformedMultipart = errors.New("malformed multipart request")
	errMissingFile        = errors.New("multipart form is missing the file field")
	errInvalidMediaType   = errors.New("media type must be photo or video")
)

const (
	errCodeInvalidRequestBody = "INVALID_REQUEST_BODY"
	errMsgInvalidRequestBody  = "invalid request body"
	errUserInvalidRequestBody = "The upload is invalid"

	errCodeMalformedMultipart = "MALFORMED_MULTIPART"
	errMsgMalformedMultipart  = "malformed multipart request"
	errUserMalformedMultipart = "The uploaded data is malformed"

	errCodeUnsupportedMediaType = "UNSUPPORTED_MEDIA_TYPE"
	errMsgUnsupportedMediaType  = "unsupported media type"
	errUserUnsupportedMediaType = "The file type is not supported"

	errCodePayloadTooLarge = "PAYLOAD_TOO_LARGE"
	errMsgPayloadTooLarge  = "media upload exceeds the size limit"
	errUserPayloadTooLarge = "The file is too large (max 10 MB)"

	errCodeInvalidExerciseID = "INVALID_EXERCISE_ID"
	errMsgInvalidExerciseID  = "invalid exercise id"
	errUserInvalidExerciseID = "The exercise ID is invalid"

	errCodeInvalidTemplateID = "INVALID_TEMPLATE_ID"
	errMsgInvalidTemplateID  = "invalid template id"
	errUserInvalidTemplateID = "The template ID is invalid"

	errCodeExerciseNotFound = "EXERCISE_NOT_FOUND"
	errUserExerciseNotFound = "Exercise not found"

	errCodeTemplateNotFound = "TEMPLATE_NOT_FOUND"
	errUserTemplateNotFound = "Template not found"

	errCodeCannotEditBuiltIn = "CANNOT_EDIT_BUILT_IN"
	errUserCannotEditBuiltIn = "Built-in exercises cannot be edited"

	errCodeForbidden = "FORBIDDEN"
	errUserForbidden = "You do not have access to this template"

	errCodeInvalidMediaType = "INVALID_MEDIA_TYPE"
	errMsgInvalidMediaType  = "media type must be photo or video"
	errUserInvalidMediaType = "Media type must be photo or video"
)

func invalidRequestBodyDetail() handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidRequestBody,
		Message:     errMsgInvalidRequestBody,
		UserMessage: errUserInvalidRequestBody,
	}
}

func malformedMultipartDetail() handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeMalformedMultipart,
		Message:     errMsgMalformedMultipart,
		UserMessage: errUserMalformedMultipart,
	}
}

func unsupportedMediaTypeDetail() handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeUnsupportedMediaType,
		Message:     errMsgUnsupportedMediaType,
		UserMessage: errUserUnsupportedMediaType,
	}
}

func payloadTooLargeDetail() handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodePayloadTooLarge,
		Message:     errMsgPayloadTooLarge,
		UserMessage: errUserPayloadTooLarge,
	}
}

func invalidExerciseIDDetail() handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidExerciseID,
		Message:     errMsgInvalidExerciseID,
		UserMessage: errUserInvalidExerciseID,
	}
}

func invalidTemplateIDDetail() handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidTemplateID,
		Message:     errMsgInvalidTemplateID,
		UserMessage: errUserInvalidTemplateID,
	}
}

func exerciseNotFoundDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeExerciseNotFound,
		Message:     err.Error(),
		UserMessage: errUserExerciseNotFound,
	}
}

func templateNotFoundDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeTemplateNotFound,
		Message:     err.Error(),
		UserMessage: errUserTemplateNotFound,
	}
}

func cannotEditBuiltInDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeCannotEditBuiltIn,
		Message:     err.Error(),
		UserMessage: errUserCannotEditBuiltIn,
	}
}

func forbiddenDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeForbidden,
		Message:     err.Error(),
		UserMessage: errUserForbidden,
	}
}

func invalidMediaTypeDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidMediaType,
		Message:     err.Error(),
		UserMessage: errUserInvalidMediaType,
	}
}
