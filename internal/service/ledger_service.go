package service

import (
	"context"
	"fmt"
	"voca-test/internal/models"
	"voca-test/internal/repository"

	"github.com/google/uuid"
)

type LedgerService struct {
	walletRepo repository.WalletRepository
	ledgerRepo repository.LedgerRepository
}

func NewLedgerService(walletRepo repository.WalletRepository, ledgerRepo repository.LedgerRepository) *LedgerService {
	return &LedgerService{
		walletRepo: walletRepo,
		ledgerRepo: ledgerRepo,
	}
}

func (s *LedgerService) GetHistory(ctx context.Context, walletID uuid.UUID) ([]models.LedgerEntry, error) {
	// Check if wallet exists first
	_, err := s.walletRepo.GetWalletByID(ctx, walletID)
	if err != nil {
		return nil, fmt.Errorf("wallet not found: %w", err)
	}

	return s.ledgerRepo.GetLedgerByWalletID(ctx, walletID)
}
