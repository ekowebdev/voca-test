package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/mock"
	"voca-test/internal/models"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) CreateUser(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// MockWalletRepository is a mock implementation of WalletRepository
type MockWalletRepository struct {
	mock.Mock
}

func (m *MockWalletRepository) CreateWallet(ctx context.Context, wallet *models.Wallet) error {
	args := m.Called(ctx, wallet)
	return args.Error(0)
}

func (m *MockWalletRepository) GetWalletByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Wallet), args.Error(1)
}

func (m *MockWalletRepository) GetWalletByIDWithLock(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*models.Wallet, error) {
	args := m.Called(ctx, tx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Wallet), args.Error(1)
}

func (m *MockWalletRepository) GetWalletByUserAndCurrency(ctx context.Context, userID uuid.UUID, currency string) (*models.Wallet, error) {
	args := m.Called(ctx, userID, currency)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Wallet), args.Error(1)
}

func (m *MockWalletRepository) UpdateWalletBalance(ctx context.Context, tx pgx.Tx, walletID uuid.UUID, newBalance interface{}) error {
	args := m.Called(ctx, tx, walletID, newBalance)
	return args.Error(0)
}

func (m *MockWalletRepository) UpdateWalletStatus(ctx context.Context, walletID uuid.UUID, status string) error {
	args := m.Called(ctx, walletID, status)
	return args.Error(0)
}

// MockLedgerRepository is a mock implementation of LedgerRepository
type MockLedgerRepository struct {
	mock.Mock
}

func (m *MockLedgerRepository) CreateLedgerEntry(ctx context.Context, tx pgx.Tx, entry *models.LedgerEntry) error {
	args := m.Called(ctx, tx, entry)
	return args.Error(0)
}

func (m *MockLedgerRepository) GetLedgerByWalletID(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]models.LedgerEntry, error) {
	args := m.Called(ctx, walletID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.LedgerEntry), args.Error(1)
}

func (m *MockLedgerRepository) CountLedgerByWalletID(ctx context.Context, walletID uuid.UUID) (int64, error) {
	args := m.Called(ctx, walletID)
	return args.Get(0).(int64), args.Error(1)
}

// MockIdempotencyRepository is a mock implementation of IdempotencyRepository
type MockIdempotencyRepository struct {
	mock.Mock
}

func (m *MockIdempotencyRepository) CheckAndCreateKey(ctx context.Context, tx pgx.Tx, key string) (bool, error) {
	args := m.Called(ctx, tx, key)
	return args.Bool(0), args.Error(1)
}
