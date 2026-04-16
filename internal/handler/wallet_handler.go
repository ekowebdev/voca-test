package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"voca-test/internal/models"
	"voca-test/internal/service"
	"voca-test/internal/util"
)

type WalletHandler struct {
	walletService *service.WalletService
}

func NewWalletHandler(s *service.WalletService) *WalletHandler {
	return &WalletHandler{walletService: s}
}

// CreateWallet - Create a new wallet
// @Summary Create a wallet for a user
// @Description Register a new wallet for a specific user with a chosen currency.
// @Tags Wallets
// @Accept json
// @Produce json
// @Param request body models.WalletCreateRequest true "Wallet creation details"
// @Success 201 {object} util.APIResponse{data=models.Wallet} "Wallet created successfully"
// @Failure 400 {object} util.APIResponse "Invalid request body or validation error"
// @Failure 500 {object} util.APIResponse "Internal server error"
// @Router /wallets [post]
func (h *WalletHandler) CreateWallet(c *gin.Context) {
	var req models.WalletCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fieldErrors := util.ParseValidationErrors(err)
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid request body", fieldErrors)
		return
	}

	wallet, err := h.walletService.CreateWallet(c.Request.Context(), req.UserID, req.Currency)
	if err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Failed to create wallet", err.Error())
		return
	}
	util.SuccessResponse(c, http.StatusCreated, wallet, "Wallet created successfully")
}

// TopUp - Top-up money
// @Summary Top-up money to a wallet
// @Description Add funds to a wallet. Requires an idempotency key to prevent duplicate transactions.
// @Tags Wallets
// @Accept json
// @Produce json
// @Param id path string true "Wallet UUID"
// @Param request body models.TopUpRequest true "Top-up details"
// @Success 200 {object} util.APIResponse{data=models.Wallet} "Top-up successful"
// @Failure 400 {object} util.APIResponse "Invalid request body or wallet ID"
// @Failure 500 {object} util.APIResponse "Internal server error"
// @Router /wallets/{id}/topup [post]
func (h *WalletHandler) TopUp(c *gin.Context) {
	idStr := c.Param("id")
	walletID, err := uuid.Parse(idStr)
	if err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid wallet ID", nil)
		return
	}

	var req models.TopUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fieldErrors := util.ParseValidationErrors(err)
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid request body", fieldErrors)
		return
	}

	wallet, err := h.walletService.TopUp(c.Request.Context(), walletID, req.Amount, req.IdempotencyKey)
	if err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Top-up failed", err.Error())
		return
	}
	util.SuccessResponse(c, http.StatusOK, wallet, "Top-up successful")
}

// Payment - Spend money
// @Summary Spend money from a wallet
// @Description Deduct funds from a wallet for a payment. Requires an idempotency key.
// @Tags Wallets
// @Accept json
// @Produce json
// @Param id path string true "Wallet UUID"
// @Param request body models.PaymentRequest true "Payment details"
// @Success 200 {object} util.APIResponse{data=models.Wallet} "Payment successful"
// @Failure 400 {object} util.APIResponse "Invalid request body, insufficient balance, or wallet ID"
// @Failure 500 {object} util.APIResponse "Internal server error"
// @Router /wallets/{id}/pay [post]
func (h *WalletHandler) Payment(c *gin.Context) {
	idStr := c.Param("id")
	walletID, err := uuid.Parse(idStr)
	if err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid wallet ID", nil)
		return
	}

	var req models.PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fieldErrors := util.ParseValidationErrors(err)
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid request body", fieldErrors)
		return
	}

	wallet, err := h.walletService.Payment(c.Request.Context(), walletID, req.Amount, req.IdempotencyKey)
	if err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Payment failed", err.Error())
		return
	}
	util.SuccessResponse(c, http.StatusOK, wallet, "Payment successful")
}

// Transfer - Move money between wallets
// @Summary Move money between same-currency wallets
// @Description Transfer funds from one wallet to another. Both wallets must use the same currency. Requires an idempotency key.
// @Tags Wallets
// @Accept json
// @Produce json
// @Param request body models.TransferRequest true "Transfer details"
// @Success 200 {object} util.APIResponse "Transfer successful"
// @Failure 400 {object} util.APIResponse "Invalid request, currency mismatch, or insufficient balance"
// @Failure 500 {object} util.APIResponse "Internal server error"
// @Router /wallets/transfer [post]
func (h *WalletHandler) Transfer(c *gin.Context) {
	var req models.TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fieldErrors := util.ParseValidationErrors(err)
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid request body", fieldErrors)
		return
	}

	err := h.walletService.Transfer(c.Request.Context(), req.FromWalletID, req.ToWalletID, req.Amount, req.IdempotencyKey)
	if err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Transfer failed", err.Error())
		return
	}
	util.SuccessResponse(c, http.StatusOK, nil, "Transfer successful")
}

// Suspend - Suspend a wallet
// @Summary Suspend a wallet
// @Description Change a wallet status to SUSPENDED. Suspended wallets cannot perform transactions.
// @Tags Wallets
// @Accept json
// @Produce json
// @Param id path string true "Wallet UUID"
// @Success 200 {object} util.APIResponse "Wallet suspended successfully"
// @Failure 400 {object} util.APIResponse "Invalid wallet ID"
// @Failure 500 {object} util.APIResponse "Internal server error"
// @Router /wallets/{id}/suspend [post]
func (h *WalletHandler) Suspend(c *gin.Context) {
	idStr := c.Param("id")
	walletID, err := uuid.Parse(idStr)
	if err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid wallet ID", nil)
		return
	}

	err = h.walletService.SuspendWallet(c.Request.Context(), walletID)
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, "Failed to suspend wallet", err.Error())
		return
	}
	util.SuccessResponse(c, http.StatusOK, nil, "Wallet suspended successfully")
}

// GetWallet - Get wallet balance and status
// @Summary Get wallet details
// @Description Retrieve the current balance and status of a specific wallet.
// @Tags Wallets
// @Accept json
// @Produce json
// @Param id path string true "Wallet UUID"
// @Success 200 {object} util.APIResponse{data=models.Wallet} "Wallet retrieved successfully"
// @Failure 400 {object} util.APIResponse "Invalid wallet ID"
// @Failure 404 {object} util.APIResponse "Wallet not found"
// @Failure 500 {object} util.APIResponse "Internal server error"
// @Router /wallets/{id} [get]
func (h *WalletHandler) GetWallet(c *gin.Context) {
	idStr := c.Param("id")
	walletID, err := uuid.Parse(idStr)
	if err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid wallet ID", nil)
		return
	}

	wallet, err := h.walletService.GetWallet(c.Request.Context(), walletID)
	if err != nil {
		util.ErrorResponse(c, http.StatusNotFound, "Wallet not found", err.Error())
		return
	}
	util.SuccessResponse(c, http.StatusOK, wallet, "Wallet retrieved successfully")
}
