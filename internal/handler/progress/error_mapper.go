package progress

import (
	"errors"
	"net/http"

	domainbodyweight "github.com/vladgrskkh/onerep-api/internal/domain/bodyweight"
	domainprogress "github.com/vladgrskkh/onerep-api/internal/domain/progress"
	"github.com/vladgrskkh/onerep-api/internal/handler"
)

const (
	errCodeInvalidRequestBody = "INVALID_REQUEST_BODY"
	errMsgInvalidRequestBody  = "invalid request body"
	errUserInvalidRequestBody = "The request body is invalid"

	errCodeInvalidExerciseID = "INVALID_EXERCISE_ID"
	errMsgInvalidExerciseID  = "invalid exercise id"
	errUserInvalidExerciseID = "The exercise ID is invalid"

	errCodeInvalidDateRange = "INVALID_DATE_RANGE"
	errMsgInvalidDateRange  = "invalid date range"
	errUserInvalidDateRange = "The from and to parameters must be RFC 3339 timestamps"

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

func invalidExerciseIDDetail() handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidExerciseID,
		Message:     errMsgInvalidExerciseID,
		UserMessage: errUserInvalidExerciseID,
	}
}

func invalidDateRangeDetail() handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidDateRange,
		Message:     errMsgInvalidDateRange,
		UserMessage: errUserInvalidDateRange,
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
	case errors.Is(err, domainprogress.ErrInvalidExerciseID):
		return http.StatusBadRequest, handler.ErrorDetail{
			Code:        errCodeInvalidExerciseID,
			Message:     err.Error(),
			UserMessage: errUserInvalidExerciseID,
		}
	case errors.Is(err, domainprogress.ErrInvalidUserID), errors.Is(err, domainbodyweight.ErrInvalidUserID):
		return http.StatusBadRequest, handler.ErrorDetail{
			Code:        "INVALID_USER_ID",
			Message:     err.Error(),
			UserMessage: "You must be logged in to view progress",
		}
	case errors.Is(err, domainbodyweight.ErrInvalidWeight):
		return http.StatusBadRequest, handler.ErrorDetail{
			Code:        "INVALID_WEIGHT",
			Message:     err.Error(),
			UserMessage: "Weight must be greater than zero",
		}
	default:
		return http.StatusInternalServerError, handler.ErrorDetail{
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
		}
	}
}
