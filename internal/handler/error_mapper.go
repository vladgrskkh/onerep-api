package handler

const (
	errCodeMissingAuthorizationHeader = "MISSING_AUTHORIZATION_HEADER"
	errMsgMissingAuthorizationHeader  = "missing authorization header"
	errUserMissingAuthorizationHeader = "You must be logged in to access this resource"

	errCodeInvalidToken = "INVALID_TOKEN"
	errMsgInvalidToken  = "invalid token"
	errUserInvalidToken = "The access token is invalid or expired"
)

// MissingAuthorizationDetail builds the 401 response for a request without an
// Authorization header or with a malformed bearer scheme.
func MissingAuthorizationDetail() ErrorDetail {
	return ErrorDetail{
		Code:        errCodeMissingAuthorizationHeader,
		Message:     errMsgMissingAuthorizationHeader,
		UserMessage: errUserMissingAuthorizationHeader,
	}
}

// InvalidTokenDetail builds the 401 response for a request whose bearer token
// failed validation.
func InvalidTokenDetail(err error) ErrorDetail {
	return ErrorDetail{
		Code:        errCodeInvalidToken,
		Message:     err.Error(),
		UserMessage: errUserInvalidToken,
	}
}
