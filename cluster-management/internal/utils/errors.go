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
)

var (
	ErrInvalidInput     = NewError(ErrCodeInvalidInput, "invalid input")
	ErrNotFound         = NewError(ErrCodeNotFound, "resource not found")
	ErrAlreadyExists    = NewError(ErrCodeAlreadyExists, "resource already exists")
	ErrInternalError    = NewError(ErrCodeInternalError, "internal server error")
	ErrValidationFailed = NewError(ErrCodeValidationFailed, "validation failed")
)
