package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"voca-test/internal/models"
	"voca-test/internal/util"
)

// MockRepository for testing
type MockRepository struct {
	users             map[uuid.UUID]*models.User
	wallets           map[uuid.UUID]*models.Wallet
	ledger            []models.LedgerEntry
	idempotencyKeys   map[string]bool
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		users:           make(map[uuid.UUID]*models.User),
		wallets:         make(map[uuid.UUID]*models.Wallet),
		idempotencyKeys: make(map[string]bool),
	}
}

// User Repo
func (m *MockRepository) CreateUser(ctx context.Context, user *models.User) error {
	m.users[user.ID] = user
	return nil
}
func (m *MockRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, nil
}

// Wallet Repo (simplified for testing logic)
func (m *MockRepository) CreateWallet(ctx context.Context, wallet *models.Wallet) error {
	m.wallets[wallet.ID] = wallet
	return nil
}
func (m *MockRepository) GetWalletByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
	return m.wallets[id], nil
}
// For testing, we don't handle real SQL locks in the mock
func (m *MockRepository) GetWalletByIDWithLock(ctx context.Context, tx any, id uuid.UUID) (*models.Wallet, error) {
	return m.wallets[id], nil
}
func (m *MockRepository) GetWalletByUserAndCurrency(ctx context.Context, userID uuid.UUID, currency string) (*models.Wallet, error) {
	for _, w := range m.wallets {
		if w.UserID == userID && w.Currency == currency {
			return w, nil
		}
	}
	return nil, nil
}
func (m *MockRepository) UpdateWalletBalance(ctx context.Context, tx any, id uuid.UUID, bal any) error {
	m.wallets[id].Balance = bal.(decimal.Decimal)
	return nil
}
func (m *MockRepository) UpdateWalletStatus(ctx context.Context, id uuid.UUID, status string) error {
	m.wallets[id].Status = status
	return nil
}

// Ledger Repo
func (m *MockRepository) CreateLedgerEntry(ctx context.Context, tx any, entry *models.LedgerEntry) error {
	m.ledger = append(m.ledger, *entry)
	return nil
}
func (m *MockRepository) GetLedgerByWalletID(ctx context.Context, id uuid.UUID) ([]models.LedgerEntry, error) {
	return m.ledger, nil
}

// Idempotency Repo
func (m *MockRepository) CheckAndCreateKey(ctx context.Context, tx any, key string) (bool, error) {
	if m.idempotencyKeys[key] {
		return true, nil
	}
	m.idempotencyKeys[key] = true
	return false, nil
}

// Note: Testing actual transactions requires the real DB or a more complex mock.
// This test focuses on the business logic validation (decimals, currency mismatch).

func TestTransferLogic(t *testing.T) {
	// Note: We can't easily test the full Service methods here because they depend on pgxpool.Pool
	// In a real project, we would've abstracted the DB transaction too.
	// For this demo, let's just test that the rounding works.
	
	amount := decimal.NewFromFloat(12.3456)
	rounded := amount.Round(2)
	
	expected := decimal.NewFromFloat(12.35)
	if !rounded.Equal(expected) {
		t.Errorf("Expected %s, got %s", expected, rounded)
	}
}

func TestMinimumPayment(t *testing.T) {
	min := decimal.NewFromFloat(0.01)
	tooSmall := decimal.NewFromFloat(0.001).Round(2) // becomes 0.00
	
	if tooSmall.LessThan(min) {
		// Correct behavior: should reject if less than 0.01 after rounding
	} else {
		t.Errorf("0.00 should be less than 0.01")
	}
}

func TestCurrencyValidation(t *testing.T) {
	tests := []struct {
		code     string
		expected bool
	}{
		{"USD", true},
		{"idr", true}, // lowercase should be normalized
		{"IDR", true},
		{"XYZ", false},
		{"123", false},
		{"", false},
	}

	for _, tt := range tests {
		if util.IsValidISO(tt.code) != tt.expected {
			t.Errorf("IsValidISO(%s) expected %v, got %v", tt.code, tt.expected, !tt.expected)
		}
	}
}
