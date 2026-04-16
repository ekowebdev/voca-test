package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"voca-test/internal/models"
	"voca-test/internal/repository"
)

func TestLedgerService_GetHistory(t *testing.T) {
	mockWalletRepo := new(repository.MockWalletRepository)
	mockLedgerRepo := new(repository.MockLedgerRepository)
	service := NewLedgerService(mockWalletRepo, mockLedgerRepo)
	ctx := context.Background()
	walletID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		mockWalletRepo.On("GetWalletByID", ctx, walletID).Return(&models.Wallet{ID: walletID}, nil).Once()
		
		expectedEntries := []models.LedgerEntry{
			{ID: uuid.New(), WalletID: walletID, Amount: decimal.NewFromInt(100), Type: "topup"},
		}
		mockLedgerRepo.On("GetLedgerByWalletID", ctx, walletID).Return(expectedEntries, nil).Once()

		entries, err := service.GetHistory(ctx, walletID)

		assert.NoError(t, err)
		assert.Len(t, entries, 1)
		assert.Equal(t, decimal.NewFromInt(100), entries[0].Amount)
		mockWalletRepo.AssertExpectations(t)
		mockLedgerRepo.AssertExpectations(t)
	})

	t.Run("Wallet Not Found", func(t *testing.T) {
		mockWalletRepo.On("GetWalletByID", ctx, walletID).Return(nil, errors.New("not found")).Once()

		entries, err := service.GetHistory(ctx, walletID)

		assert.Error(t, err)
		assert.Nil(t, entries)
		assert.Contains(t, err.Error(), "wallet not found")
		mockWalletRepo.AssertExpectations(t)
	})

	t.Run("Ledger Repository Error", func(t *testing.T) {
		mockWalletRepo.On("GetWalletByID", ctx, walletID).Return(&models.Wallet{ID: walletID}, nil).Once()
		mockLedgerRepo.On("GetLedgerByWalletID", ctx, walletID).Return(nil, errors.New("db error")).Once()

		entries, err := service.GetHistory(ctx, walletID)

		assert.Error(t, err)
		assert.Nil(t, entries)
		assert.Contains(t, err.Error(), "db error")
		mockWalletRepo.AssertExpectations(t)
		mockLedgerRepo.AssertExpectations(t)
	})
}
