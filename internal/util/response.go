package util

import (
	"github.com/gin-gonic/gin"
)

// APIResponse is the standard structure for all API responses
type APIResponse struct {
	Status  string      `json:"status"`            // "success" or "error"
	Message string      `json:"message,omitempty"` // General message
	Data    interface{} `json:"data,omitempty"`    // Actual payload for success
	Errors  interface{} `json:"errors,omitempty"`  // Specific error details (e.g., validation)
}

// SuccessResponse sends a standard success response
func SuccessResponse(c *gin.Context, statusCode int, data interface{}, message string) {
	c.JSON(statusCode, APIResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

// ErrorResponse sends a standard error response
func ErrorResponse(c *gin.Context, statusCode int, message string, errors interface{}) {
	c.JSON(statusCode, APIResponse{
		Status:  "error",
		Message: message,
		Errors:  errors,
	})
}
