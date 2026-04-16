package routes

import (
	"github.com/gin-gonic/gin"
	"voca-test/internal/handler"
)

// RegisterUserRoutes registers all user-related routes
func RegisterUserRoutes(rg *gin.RouterGroup, h *handler.UserHandler) {
	rg.POST("/users", h.CreateUser)
}
