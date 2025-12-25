package utils

import "fmt"

type Error struct {
	Code    int
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("error %d: %s - %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("error %d: %s", e.Code, e.Message)
}

func NewError(code int, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

func WrapError(code int, message string, err error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

const (
	ErrCodeInvalidInput     = 1001
	ErrCodeNotFound         = 1002
	ErrCodeAlreadyExists    = 1003
	ErrCodeInternalError    = 1004
	ErrCodeValidationFailed = 1005
	ErrCodeUnauthorized     = 1006
	ErrCodeForbidden        = 1007
	ErrCodeConflict         = 1008
	ErrCodeBadRequest       = 1009
	ErrCodeServiceUnavailable = 1010
	ErrCodeTimeout          = 1011
	ErrCodeQuotaExceeded    = 1012
	ErrCodeInvalidState     = 1013
)

var (
	ErrInvalidInput     = NewError(ErrCodeInvalidInput, "invalid input")
	ErrNotFound         = NewError(ErrCodeNotFound, "resource not found")
	ErrAlreadyExists    = NewError(ErrCodeAlreadyExists, "resource already exists")
	ErrInternalError    = NewError(ErrCodeInternalError, "internal server error")
	ErrValidationFailed = NewError(ErrCodeValidationFailed, "validation failed")
	ErrUnauthorized     = NewError(ErrCodeUnauthorized, "unauthorized")
	ErrForbidden        = NewError(ErrCodeForbidden, "forbidden")
	ErrConflict         = NewError(ErrCodeConflict, "conflict")
	ErrBadRequest       = NewError(ErrCodeBadRequest, "bad request")
	ErrServiceUnavailable = NewError(ErrCodeServiceUnavailable, "service unavailable")
	ErrTimeout          = NewError(ErrCodeTimeout, "operation timeout")
	ErrQuotaExceeded    = NewError(ErrCodeQuotaExceeded, "quota exceeded")
	ErrInvalidState     = NewError(ErrCodeInvalidState, "invalid state")
)

func GetHTTPStatusCode(errCode int) int {
	switch errCode {
	case ErrCodeInvalidInput, ErrCodeValidationFailed, ErrCodeBadRequest:
		return 400
	case ErrCodeUnauthorized:
		return 401
	case ErrCodeForbidden:
		return 403
	case ErrCodeNotFound:
		return 404
	case ErrCodeConflict, ErrCodeAlreadyExists:
		return 409
	case ErrCodeQuotaExceeded:
		return 429
	case ErrCodeServiceUnavailable:
		return 503
	case ErrCodeTimeout:
		return 504
	case ErrCodeInternalError:
		return 500
	default:
		return 500
	}
}
