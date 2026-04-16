package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Wallet represents a user's currency-specific wallet
type Wallet struct {
	ID        uuid.UUID       `json:"id"`
	UserID    uuid.UUID       `json:"user_id"`
	Currency  string          `json:"currency" binding:"required,len=3"`
	Balance   decimal.Decimal `json:"balance"`
	Status    string          `json:"status"` // ACTIVE, SUSPENDED
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// Transaction Types
const (
	TypeTopUp       = "TOPUP"
	TypePayment     = "PAYMENT"
	TypeTransferIn  = "TRANSFER_IN"
	TypeTransferOut = "TRANSFER_OUT"
)

// Wallet Statuses
const (
	StatusActive    = "ACTIVE"
	StatusSuspended = "SUSPENDED"
)

// Request Structures
type WalletCreateRequest struct {
	UserID   uuid.UUID `json:"user_id" binding:"required"`
	Currency string    `json:"currency" binding:"required,len=3"`
}

type TopUpRequest struct {
	Amount         decimal.Decimal `json:"amount" binding:"required"`
	IdempotencyKey string          `json:"idempotency_key" binding:"required"`
}

type PaymentRequest struct {
	Amount         decimal.Decimal `json:"amount" binding:"required"`
	IdempotencyKey string          `json:"idempotency_key" binding:"required"`
}

type TransferRequest struct {
	FromWalletID   uuid.UUID       `json:"from_wallet_id" binding:"required"`
	ToWalletID     uuid.UUID       `json:"to_wallet_id" binding:"required"`
	Amount         decimal.Decimal `json:"amount" binding:"required"`
	IdempotencyKey string          `json:"idempotency_key" binding:"required"`
}
