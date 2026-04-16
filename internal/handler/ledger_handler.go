package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "voca-test/internal/models"
	"voca-test/internal/service"
	"voca-test/internal/util"
)

type LedgerHandler struct {
	ledgerService *service.LedgerService
}

func NewLedgerHandler(s *service.LedgerService) *LedgerHandler {
	return &LedgerHandler{ledgerService: s}
}

// GetHistory - Get transaction history
// @Summary Get wallet transaction history
// @Description Retrieve a list of all transactions (ledger entries) for a specific wallet, sorted by date.
// @Tags Wallets
// @Accept json
// @Produce json
// @Param id path string true "Wallet UUID"
// @Success 200 {object} util.APIResponse{data=[]models.LedgerEntry} "Transaction history retrieved successfully"
// @Failure 400 {object} util.APIResponse "Invalid wallet ID"
// @Failure 500 {object} util.APIResponse "Internal server error"
// @Router /wallets/{id}/transactions [get]
func (h *LedgerHandler) GetHistory(c *gin.Context) {
	idStr := c.Param("id")
	walletID, err := uuid.Parse(idStr)
	if err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid wallet ID", nil)
		return
	}

	history, err := h.ledgerService.GetHistory(c.Request.Context(), walletID)
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch history", err.Error())
		return
	}

	util.SuccessResponse(c, http.StatusOK, history, "Transaction history retrieved successfully")
}
