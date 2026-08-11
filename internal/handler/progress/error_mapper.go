package progress

import "github.com/vladgrskkh/onerep-api/internal/handler"

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

	errCodeInvalidUserID = "INVALID_USER_ID"
	errUserInvalidUserID = "You must be logged in to view progress"

	errCodeInvalidWeight = "INVALID_WEIGHT"
	errUserInvalidWeight = "Weight must be greater than zero"
)

func invalidRequestBodyDetail() handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidRequestBody,
		Message:     errMsgInvalidRequestBody,
		UserMessage: errUserInvalidRequestBody,
	}
}

func invalidExerciseIDDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidExerciseID,
		Message:     err.Error(),
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

func invalidUserIDDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidUserID,
		Message:     err.Error(),
		UserMessage: errUserInvalidUserID,
	}
}

func invalidWeightDetail(err error) handler.ErrorDetail {
	return handler.ErrorDetail{
		Code:        errCodeInvalidWeight,
		Message:     err.Error(),
		UserMessage: errUserInvalidWeight,
	}
}
