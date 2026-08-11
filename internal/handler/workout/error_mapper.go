package workout

import "github.com/vladgrskkh/onerep-api/internal/handler"

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

	errCodeWorkoutNotFound = "WORKOUT_NOT_FOUND"
	errUserWorkoutNotFound = "Workout not found"

	errCodeWorkoutExerciseNotFound = "WORKOUT_EXERCISE_NOT_FOUND"
	errUserWorkoutExerciseNotFound = "Workout exercise not found"

	errCodeActiveWorkout = "ACTIVE_WORKOUT_EXISTS"
	errUserActiveWorkout = "You already have an active workout"

	errCodeTemplateNotFound = "TEMPLATE_NOT_FOUND"
	errUserTemplateNotFound = "Template not found"

	errCodeInvalidUserID = "INVALID_USER_ID"
	errUserInvalidUserID = "You must be logged in to manage workouts"

	errCodeInvalidExerciseID = "INVALID_EXERCISE_ID"
	errUserInvalidExerciseID = "The exercise ID is invalid"

	errCodeInvalidWeight = "INVALID_WEIGHT"
	errUserInvalidWeight = "Weight must be greater than zero"

	errCodeInvalidReps = "INVALID_REPS"
	errUserInvalidReps = "Reps must be greater than zero"

	errCodeInvalidRPE = "INVALID_RPE"
	errUserInvalidRPE = "RPE must be between 1 and 10"

	errCodeInvalidRestSeconds = "INVALID_REST_SECONDS"
	errUserInvalidRestSeconds = "Rest seconds must not be negative"
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

func workoutNotFoundDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeWorkoutNotFound,
		Message:     err.Error(),
		UserMessage: errUserWorkoutNotFound,
	}
}

func workoutExerciseNotFoundDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeWorkoutExerciseNotFound,
		Message:     err.Error(),
		UserMessage: errUserWorkoutExerciseNotFound,
	}
}

func activeWorkoutDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeActiveWorkout,
		Message:     err.Error(),
		UserMessage: errUserActiveWorkout,
	}
}

func templateNotFoundDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeTemplateNotFound,
		Message:     err.Error(),
		UserMessage: errUserTemplateNotFound,
	}
}

func invalidUserIDDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidUserID,
		Message:     err.Error(),
		UserMessage: errUserInvalidUserID,
	}
}

func invalidExerciseIDDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidExerciseID,
		Message:     err.Error(),
		UserMessage: errUserInvalidExerciseID,
	}
}

func invalidWeightDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidWeight,
		Message:     err.Error(),
		UserMessage: errUserInvalidWeight,
	}
}

func invalidRepsDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidReps,
		Message:     err.Error(),
		UserMessage: errUserInvalidReps,
	}
}

func invalidRPEDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidRPE,
		Message:     err.Error(),
		UserMessage: errUserInvalidRPE,
	}
}

func invalidRestSecondsDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidRestSeconds,
		Message:     err.Error(),
		UserMessage: errUserInvalidRestSeconds,
	}
}
