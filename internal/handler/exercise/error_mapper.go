package exercise

import "github.com/vladgrskkh/onerep-api/internal/handler"

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

	errCodeExerciseNotFound = "EXERCISE_NOT_FOUND"
	errUserExerciseNotFound = "Exercise not found"

	errCodeMuscleGroupNotFound = "MUSCLE_GROUP_NOT_FOUND"
	errUserMuscleGroupNotFound = "One or more muscle groups do not exist"

	errCodeCannotEditBuiltIn = "CANNOT_EDIT_BUILT_IN"
	errUserCannotEditBuiltIn = "Built-in exercises cannot be edited"

	errCodeInvalidName = "INVALID_NAME"
	errUserInvalidName = "Name must not be empty"

	errCodeInvalidUserID = "INVALID_USER_ID"
	errUserInvalidUserID = "You must be logged in to create exercises"

	errCodeInvalidMuscleGroupID = "INVALID_MUSCLE_GROUP_ID"
	errUserInvalidMuscleGroupID = "Muscle group IDs must be positive"
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

func exerciseNotFoundDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeExerciseNotFound,
		Message:     err.Error(),
		UserMessage: errUserExerciseNotFound,
	}
}

func muscleGroupNotFoundDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeMuscleGroupNotFound,
		Message:     err.Error(),
		UserMessage: errUserMuscleGroupNotFound,
	}
}

func cannotEditBuiltInDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeCannotEditBuiltIn,
		Message:     err.Error(),
		UserMessage: errUserCannotEditBuiltIn,
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

func invalidMuscleGroupIDDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidMuscleGroupID,
		Message:     err.Error(),
		UserMessage: errUserInvalidMuscleGroupID,
	}
}
