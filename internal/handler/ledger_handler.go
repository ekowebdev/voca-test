package handler

import (
	"net/http"
	"strconv"

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
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(10)
// @Param type query string false "Transaction type filter (TOPUP, PAYMENT, TRANSFER_IN, TRANSFER_OUT)"
// @Success 200 {object} util.APIResponse{data=[]models.LedgerEntry,meta=util.PaginationMeta} "Transaction history retrieved successfully"
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

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))
	txType := c.Query("type")

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}

	history, meta, summary, err := h.ledgerService.GetHistory(c.Request.Context(), walletID, txType, page, perPage)
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch history", err.Error())
		return
	}

	meta.GenerateLinks(c)

	util.SuccessResponseWithPagination(c, http.StatusOK, history, *meta, summary, "Transaction history retrieved successfully")
}
