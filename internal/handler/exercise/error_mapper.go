package exercise

import (
	"errors"
	"net/http"

	domainexercise "github.com/vladgrskkh/onerep-api/internal/domain/exercise"
	"github.com/vladgrskkh/onerep-api/internal/handler"
)

const (
	errCodeInvalidRequestBody = "INVALID_REQUEST_BODY"
	errMsgInvalidRequestBody  = "invalid request body"
	errUserInvalidRequestBody = "The request body is invalid"

	errCodeInvalidExerciseID = "INVALID_EXERCISE_ID"
	errMsgInvalidExerciseID  = "invalid exercise id"
	errUserInvalidExerciseID = "The exercise ID is invalid"

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

func invalidSinceDetail() handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidSince,
		Message:     errMsgInvalidSince,
		UserMessage: errUserInvalidSince,
	}
}

func mapError(err error) (int, handler.ErrorDetail) {
	switch {
	case errors.Is(err, domainexercise.ErrExerciseNotFound):
		return http.StatusNotFound, handler.ErrorDetail{
			Code:        "EXERCISE_NOT_FOUND",
			Message:     err.Error(),
			UserMessage: "Exercise not found",
		}
	case errors.Is(err, domainexercise.ErrMuscleGroupMissing):
		return http.StatusNotFound, handler.ErrorDetail{
			Code:        "MUSCLE_GROUP_NOT_FOUND",
			Message:     err.Error(),
			UserMessage: "One or more muscle groups do not exist",
		}
	case errors.Is(err, domainexercise.ErrCannotEditBuiltIn):
		return http.StatusBadRequest, handler.ErrorDetail{
			Code:        "CANNOT_EDIT_BUILT_IN",
			Message:     err.Error(),
			UserMessage: "Built-in exercises cannot be edited",
		}
	case errors.Is(err, domainexercise.ErrInvalidName):
		return http.StatusBadRequest, handler.ErrorDetail{
			Code:        "INVALID_NAME",
			Message:     err.Error(),
			UserMessage: "Name must not be empty",
		}
	case errors.Is(err, domainexercise.ErrInvalidUserID):
		return http.StatusBadRequest, handler.ErrorDetail{
			Code:        "INVALID_USER_ID",
			Message:     err.Error(),
			UserMessage: "You must be logged in to create exercises",
		}
	case errors.Is(err, domainexercise.ErrInvalidID):
		return http.StatusBadRequest, handler.ErrorDetail{
			Code:        "INVALID_MUSCLE_GROUP_ID",
			Message:     err.Error(),
			UserMessage: "Muscle group IDs must be positive",
		}
	default:
		return http.StatusInternalServerError, handler.ErrorDetail{
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
		}
	}
}
