package routes

import (
	"net/http"

	"voca-test/internal/handler"
	"voca-test/internal/middleware"
	"voca-test/internal/util"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "voca-test/docs"
)

// SetupRoutes initializes all the routes for the application
func SetupRoutes(
	r *gin.Engine,
	cfg *util.Config,
	userHandler *handler.UserHandler,
	walletHandler *handler.WalletHandler,
	ledgerHandler *handler.LedgerHandler,
) {
	// Register Middlewares
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS(cfg.Origins))
	r.Use(middleware.SecurityHeaders())

	// Standard response for 404 - Not Found
	r.NoRoute(func(c *gin.Context) {
		util.ErrorResponse(c, http.StatusNotFound, "Resource not found", nil)
	})

	// Standard response for 405 - Method Not Allowed
	r.NoMethod(func(c *gin.Context) {
		util.ErrorResponse(c, http.StatusMethodNotAllowed, "Method not allowed", nil)
	})

	// Health check
	r.GET("/health", func(c *gin.Context) {
		util.SuccessResponse(c, http.StatusOK, gin.H{"status": "up"}, "System health status")
	})

	// Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Domain Routes
	v1 := r.Group("/api/v1")
	{
		RegisterUserRoutes(v1, userHandler)
		RegisterWalletRoutes(v1, walletHandler)
		RegisterLedgerRoutes(v1, ledgerHandler)
	}
}
