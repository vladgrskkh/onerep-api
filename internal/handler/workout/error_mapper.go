package workout

import (
	"errors"
	"net/http"

	domaintemplate "github.com/vladgrskkh/onerep-api/internal/domain/template"
	domainworkout "github.com/vladgrskkh/onerep-api/internal/domain/workout"
	"github.com/vladgrskkh/onerep-api/internal/handler"
)

const (
	errCodeInvalidRequestBody = "INVALID_REQUEST_BODY"
	errMsgInvalidRequestBody  = "invalid request body"
	errUserInvalidRequestBody = "The request body is invalid"

	errCodeInvalidWorkoutID = "INVALID_WORKOUT_ID"
	errMsgInvalidWorkoutID  = "invalid workout id"
	errUserInvalidWorkoutID = "The workout ID is invalid"

	errCodeInvalidWorkoutExerciseID = "INVALID_WORKOUT_EXERCISE_ID"
	errMsgInvalidWorkoutExerciseID  = "invalid workout exercise id"
	errUserInvalidWorkoutExerciseID = "The workout exercise ID is invalid"

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

func invalidWorkoutIDDetail() handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidWorkoutID,
		Message:     errMsgInvalidWorkoutID,
		UserMessage: errUserInvalidWorkoutID,
	}
}

func invalidWorkoutExerciseIDDetail() handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidWorkoutExerciseID,
		Message:     errMsgInvalidWorkoutExerciseID,
		UserMessage: errUserInvalidWorkoutExerciseID,
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
	case errors.Is(err, domainworkout.ErrWorkoutNotFound):
		return http.StatusNotFound, handler.ErrorDetail{
			Code:        "WORKOUT_NOT_FOUND",
			Message:     err.Error(),
			UserMessage: "Workout not found",
		}
	case errors.Is(err, domainworkout.ErrWorkoutExerciseNotFound):
		return http.StatusNotFound, handler.ErrorDetail{
			Code:        "WORKOUT_EXERCISE_NOT_FOUND",
			Message:     err.Error(),
			UserMessage: "Workout exercise not found",
		}
	case errors.Is(err, domainworkout.ErrActiveWorkout):
		return http.StatusConflict, handler.ErrorDetail{
			Code:        "ACTIVE_WORKOUT_EXISTS",
			Message:     err.Error(),
			UserMessage: "You already have an active workout",
		}
	case errors.Is(err, domaintemplate.ErrTemplateNotFound):
		return http.StatusNotFound, handler.ErrorDetail{
			Code:        "TEMPLATE_NOT_FOUND",
			Message:     err.Error(),
			UserMessage: "Template not found",
		}
	case errors.Is(err, domainworkout.ErrInvalidUserID):
		return http.StatusBadRequest, handler.ErrorDetail{
			Code:        "INVALID_USER_ID",
			Message:     err.Error(),
			UserMessage: "You must be logged in to manage workouts",
		}
	case errors.Is(err, domainworkout.ErrInvalidWorkoutID):
		return http.StatusBadRequest, handler.ErrorDetail{
			Code:        "INVALID_WORKOUT_ID",
			Message:     err.Error(),
			UserMessage: errUserInvalidWorkoutID,
		}
	case errors.Is(err, domainworkout.ErrInvalidExerciseID):
		return http.StatusBadRequest, handler.ErrorDetail{
			Code:        "INVALID_EXERCISE_ID",
			Message:     err.Error(),
			UserMessage: "The exercise ID is invalid",
		}
	case errors.Is(err, domainworkout.ErrInvalidWorkoutExerciseID):
		return http.StatusBadRequest, handler.ErrorDetail{
			Code:        "INVALID_WORKOUT_EXERCISE_ID",
			Message:     err.Error(),
			UserMessage: errUserInvalidWorkoutExerciseID,
		}
	case errors.Is(err, domainworkout.ErrInvalidWeight):
		return http.StatusBadRequest, handler.ErrorDetail{
			Code:        "INVALID_WEIGHT",
			Message:     err.Error(),
			UserMessage: "Weight must be greater than zero",
		}
	case errors.Is(err, domainworkout.ErrInvalidReps):
		return http.StatusBadRequest, handler.ErrorDetail{
			Code:        "INVALID_REPS",
			Message:     err.Error(),
			UserMessage: "Reps must be greater than zero",
		}
	case errors.Is(err, domainworkout.ErrInvalidRPE):
		return http.StatusBadRequest, handler.ErrorDetail{
			Code:        "INVALID_RPE",
			Message:     err.Error(),
			UserMessage: "RPE must be between 1 and 10",
		}
	case errors.Is(err, domainworkout.ErrInvalidRestSeconds):
		return http.StatusBadRequest, handler.ErrorDetail{
			Code:        "INVALID_REST_SECONDS",
			Message:     err.Error(),
			UserMessage: "Rest seconds must not be negative",
		}
	default:
		return http.StatusInternalServerError, handler.ErrorDetail{
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
		}
	}
}
