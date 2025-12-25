package utils

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/taichu-system/cluster-management/internal/utils"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func Success(c *gin.Context, statusCode int, data interface{}) {
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.JSON(statusCode, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

func Error(c *gin.Context, errCode int, format string, args ...interface{}) {
	c.Header("Content-Type", "application/json; charset=utf-8")
	message := format
	if len(args) > 0 {
		message = fmt.Sprintf(format, args...)
	}

	statusCode := utils.GetHTTPStatusCode(errCode)
	c.JSON(statusCode, Response{
		Code:    errCode,
		Message: message,
		Data:    nil,
	})
}

func ErrorWithStatus(c *gin.Context, statusCode int, errCode int, format string, args ...interface{}) {
	c.Header("Content-Type", "application/json; charset=utf-8")
	message := format
	if len(args) > 0 {
		message = fmt.Sprintf(format, args...)
	}

	c.JSON(statusCode, Response{
		Code:    errCode,
		Message: message,
		Data:    nil,
	})
}

func HandleError(c *gin.Context, err error, defaultFormat string, args ...interface{}) {
	if customErr, ok := err.(*utils.Error); ok {
		statusCode := utils.GetHTTPStatusCode(customErr.Code)
		message := customErr.Message
		if len(args) > 0 {
			message = fmt.Sprintf(defaultFormat, args...)
		}
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.JSON(statusCode, Response{
			Code:    customErr.Code,
			Message: message,
			Data:    nil,
		})
		return
	}

	Error(c, utils.ErrCodeInternalError, defaultFormat, args...)
}
