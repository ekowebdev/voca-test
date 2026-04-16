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
	txType := ""

	t.Run("Success", func(t *testing.T) {
		mockWalletRepo.On("GetWalletByID", ctx, walletID).Return(&models.Wallet{ID: walletID}, nil).Once()
		mockLedgerRepo.On("CountLedgerByWalletID", ctx, walletID, txType).Return(int64(1), nil).Once()

		expectedEntries := []models.LedgerEntry{
			{ID: uuid.New(), WalletID: walletID, Amount: decimal.NewFromInt(100), Type: "topup"},
		}
		mockLedgerRepo.On("GetLedgerByWalletID", ctx, walletID, txType, 10, 0).Return(expectedEntries, nil).Once()

		expectedSummary := map[string]interface{}{"total_credit": decimal.NewFromInt(100), "total_debit": decimal.Zero}
		mockLedgerRepo.On("GetLedgerSummary", ctx, walletID, txType).Return(expectedSummary, nil).Once()

		entries, meta, summary, err := service.GetHistory(ctx, walletID, txType, 1, 10)

		assert.NoError(t, err)
		assert.Len(t, entries, 1)
		assert.NotNil(t, meta)
		assert.NotNil(t, summary)
		assert.Equal(t, decimal.NewFromInt(100), summary["total_credit"])
		assert.Equal(t, 1, meta.CurrentPage)
		assert.Equal(t, int64(1), meta.TotalItems)
		assert.Equal(t, decimal.NewFromInt(100), entries[0].Amount)
		mockWalletRepo.AssertExpectations(t)
		mockLedgerRepo.AssertExpectations(t)
	})

	t.Run("Wallet Not Found", func(t *testing.T) {
		mockWalletRepo.On("GetWalletByID", ctx, walletID).Return(nil, errors.New("not found")).Once()

		entries, meta, summary, err := service.GetHistory(ctx, walletID, txType, 1, 10)

		assert.Error(t, err)
		assert.Nil(t, entries)
		assert.Nil(t, meta)
		assert.Nil(t, summary)
		assert.Contains(t, err.Error(), "wallet not found")
		mockWalletRepo.AssertExpectations(t)
	})

	t.Run("Ledger Repository Error", func(t *testing.T) {
		mockWalletRepo.On("GetWalletByID", ctx, walletID).Return(&models.Wallet{ID: walletID}, nil).Once()
		mockLedgerRepo.On("CountLedgerByWalletID", ctx, walletID, txType).Return(int64(1), nil).Once()
		mockLedgerRepo.On("GetLedgerByWalletID", ctx, walletID, txType, 10, 0).Return(nil, errors.New("db error")).Once()

		entries, meta, summary, err := service.GetHistory(ctx, walletID, txType, 1, 10)

		assert.Error(t, err)
		assert.Nil(t, entries)
		assert.Nil(t, meta)
		assert.Nil(t, summary)
		assert.Contains(t, err.Error(), "db error")
		mockWalletRepo.AssertExpectations(t)
		mockLedgerRepo.AssertExpectations(t)
	})
}
