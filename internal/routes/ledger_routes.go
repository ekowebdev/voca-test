package routes

import (
	"github.com/gin-gonic/gin"
	"voca-test/internal/handler"
)

// RegisterLedgerRoutes registers all ledger-related routes
func RegisterLedgerRoutes(rg *gin.RouterGroup, h *handler.LedgerHandler) {
	wallets := rg.Group("/wallets")
	{
		wallets.GET("/:id/transactions", h.GetHistory)
	}
}
