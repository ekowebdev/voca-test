package routes

import (
	"github.com/gin-gonic/gin"
	"voca-test/internal/handler"
)

// RegisterWalletRoutes registers all wallet-related routes
func RegisterWalletRoutes(rg *gin.RouterGroup, h *handler.WalletHandler) {
	wallets := rg.Group("/wallets")
	{
		wallets.POST("", h.CreateWallet)
		wallets.GET("/:id", h.GetWallet)
		wallets.POST("/:id/topup", h.TopUp)
		wallets.POST("/:id/pay", h.Payment)
		wallets.POST("/:id/suspend", h.Suspend)
		wallets.POST("/transfer", h.Transfer)
	}
}
