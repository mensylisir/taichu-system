package utils

import (
	"strconv"

	"github.com/google/uuid"
)

// ParseInt 解析整数，如果解析失败则返回默认值
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

// ParseUUID 解析UUID，如果解析失败则返回零值和错误
func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// MustParseUUID 解析UUID，如果解析失败则panic
func MustParseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		panic(err)
	}
	return id
}
