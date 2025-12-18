package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func Success(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

func Error(c *gin.Context, statusCode int, format string, args ...interface{}) {
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
