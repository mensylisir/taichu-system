package utils

import (
	"github.com/gin-gonic/gin"
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

func Error(c *gin.Context, statusCode int, format string, args ...interface{}) {
	c.Header("Content-Type", "application/json; charset=utf-8")
	message := format
	if len(args) > 0 {
		message = format
	}

	c.JSON(statusCode, Response{
		Code:    -1,
		Message: message,
		Data:    nil,
	})
}
