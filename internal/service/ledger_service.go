package service

import (
	"context"
	"fmt"
	"math"
	"voca-test/internal/models"
	"voca-test/internal/repository"
	"voca-test/internal/util"

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

func (s *LedgerService) GetHistory(ctx context.Context, walletID uuid.UUID, page, perPage int) ([]models.LedgerEntry, *util.PaginationMeta, error) {
	// Check if wallet exists first
	_, err := s.walletRepo.GetWalletByID(ctx, walletID)
	if err != nil {
		return nil, nil, fmt.Errorf("wallet not found: %w", err)
	}

	totalItems, err := s.ledgerRepo.CountLedgerByWalletID(ctx, walletID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to count history: %w", err)
	}

	offset := (page - 1) * perPage
	history, err := s.ledgerRepo.GetLedgerByWalletID(ctx, walletID, perPage, offset)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch history: %w", err)
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(perPage)))

	meta := &util.PaginationMeta{
		CurrentPage: page,
		PerPage:     perPage,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
	}

	return history, meta, nil
}
