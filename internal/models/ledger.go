package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// LedgerEntry represents a single transaction record in the ledger
type LedgerEntry struct {
	ID           uuid.UUID       `json:"id"`
	WalletID     uuid.UUID       `json:"wallet_id"`
	Amount       decimal.Decimal `json:"amount"`
	Type         string          `json:"type"` // TOPUP, PAYMENT, TRANSFER_IN, TRANSFER_OUT
	BalanceAfter decimal.Decimal `json:"balance_after"`
	ReferenceID  *uuid.UUID      `json:"reference_id,omitempty"`
	Description  string          `json:"description"`
	CreatedAt    time.Time       `json:"created_at"`
}
