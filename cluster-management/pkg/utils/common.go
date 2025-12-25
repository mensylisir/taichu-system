package utils

import (
	"strconv"

	"github.com/google/uuid"
)

const (
	ErrCodeInvalidInput       = 1001
	ErrCodeNotFound           = 1002
	ErrCodeAlreadyExists      = 1003
	ErrCodeInternalError      = 1004
	ErrCodeValidationFailed   = 1005
	ErrCodeUnauthorized       = 1006
	ErrCodeForbidden          = 1007
	ErrCodeConflict           = 1008
	ErrCodeBadRequest         = 1009
	ErrCodeServiceUnavailable = 1010
	ErrCodeTimeout            = 1011
	ErrCodeQuotaExceeded      = 1012
	ErrCodeInvalidState       = 1013
)

func ParseInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	result, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	return result
}

func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
