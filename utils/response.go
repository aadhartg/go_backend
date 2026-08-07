package utils

import (
	"github.com/gin-gonic/gin"
)

type Response struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"` // Will omit if Data is nil
}

func JSONResponse(c *gin.Context, status int, message string, data interface{}) {
	if data == nil {
		data = struct{}{}
	}

	response := Response{
		Message: message,
		Data:    data,
	}

	c.JSON(status, response)
}
