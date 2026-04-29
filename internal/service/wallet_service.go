package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"voca-test/internal/models"
	"voca-test/internal/repository"
	"voca-test/internal/util"
)

const walletLockShards = 64

type WalletService struct {
	userRepo repository.UserRepository
	wRepo    repository.WalletRepository
	lRepo    repository.LedgerRepository
	iRepo    repository.IdempotencyRepository
	db       *pgxpool.Pool
	mu       [walletLockShards]sync.Mutex
}

func NewWalletService(
	pool *pgxpool.Pool,
	userRepo repository.UserRepository,
	walletRepo repository.WalletRepository,
	ledgerRepo repository.LedgerRepository,
	idempotencyRepo repository.IdempotencyRepository,
) *WalletService {
	return &WalletService{
		userRepo: userRepo,
		wRepo:    walletRepo,
		lRepo:    ledgerRepo,
		iRepo:    idempotencyRepo,
		db:       pool,
	}
}

// getShard returns the shard index for a given UUID
func (s *WalletService) getShard(id uuid.UUID) int {
	// Simple hash by using the first 8 bytes of UUID
	var sum uint64
	for i := 0; i < 8; i++ {
		sum = (sum << 8) | uint64(id[i])
	}
	return int(sum % uint64(walletLockShards))
}

// CreateWallet handles creating a new wallet for a user
func (s *WalletService) CreateWallet(ctx context.Context, userID uuid.UUID, currency string) (*models.Wallet, error) {
	shardIdx := s.getShard(userID)
	s.mu[shardIdx].Lock()
	defer s.mu[shardIdx].Unlock()

	// Normalize currency to uppercase
	currency = strings.ToUpper(strings.TrimSpace(currency))

	// 1. Validate ISO Currency Code
	if !util.IsValidISO(currency) {
		return nil, fmt.Errorf("invalid ISO currency code: %s", currency)
	}

	// Check if user exists
	_, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user: %w", err)
	}

	// Double check if wallet already exists for this currency
	existing, _ := s.wRepo.GetWalletByUserAndCurrency(ctx, userID, currency)
	if existing != nil {
		return nil, fmt.Errorf("wallet for currency %s already exists for this user", currency)
	}

	wallet := &models.Wallet{
		ID:        uuid.New(),
		UserID:    userID,
		Currency:  currency,
		Balance:   decimal.Zero,
		Status:    models.StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.wRepo.CreateWallet(ctx, wallet); err != nil {
		return nil, err
	}
	return wallet, nil
}

// TopUp handles adding money to a wallet
func (s *WalletService) TopUp(ctx context.Context, walletID uuid.UUID, amount decimal.Decimal, idempotencyKey string) (*models.Wallet, error) {
	shardIdx := s.getShard(walletID)
	s.mu[shardIdx].Lock()
	defer s.mu[shardIdx].Unlock()

	// 1. Minimum unit check and rounding
	// Round to 2 decimal places as per requirement
	roundedAmount := amount.Round(2)
	if roundedAmount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("top-up amount must be positive")
	}

	// Start Transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// 2. Idempotency Check
	cachedCode, cachedBody, found, err := s.iRepo.GetResponse(ctx, idempotencyKey)
	if err == nil && found {
		if cachedCode >= 200 && cachedCode < 300 {
			var cachedWallet models.Wallet
			if err := json.Unmarshal([]byte(cachedBody), &cachedWallet); err == nil {
				return &cachedWallet, nil
			}
		}
		return nil, fmt.Errorf("duplicate request: idempotency key %s already used", idempotencyKey)
	}

	isDup, err := s.iRepo.CheckAndCreateKey(ctx, tx, idempotencyKey)
	if err != nil {
		return nil, err
	}
	if isDup {
		return nil, fmt.Errorf("duplicate request: idempotency key %s already used", idempotencyKey)
	}

	// 3. Fetch Wallet with Pessimistic Lock (SELECT FOR UPDATE)
	wallet, err := s.wRepo.GetWalletByIDWithLock(ctx, tx, walletID)
	if err != nil {
		return nil, err
	}

	if wallet.Status == models.StatusSuspended {
		return nil, errors.New("wallet is suspended")
	}

	// 4. Update Balance
	newBalance := wallet.Balance.Add(roundedAmount)
	if err := s.wRepo.UpdateWalletBalance(ctx, tx, walletID, newBalance); err != nil {
		return nil, err
	}

	// 5. Audit Ledger
	entry := &models.LedgerEntry{
		ID:           uuid.New(),
		WalletID:     walletID,
		Amount:       roundedAmount,
		Type:         models.TypeTopUp,
		BalanceAfter: newBalance,
		Description:  fmt.Sprintf("Top-up of %s %s", roundedAmount, wallet.Currency),
		CreatedAt:    time.Now(),
	}
	if err := s.lRepo.CreateLedgerEntry(ctx, tx, entry); err != nil {
		return nil, err
	}

	// 6. Save Response for Idempotency
	wallet.Balance = newBalance
	respBody, _ := json.Marshal(wallet)
	if err := s.iRepo.SaveResponse(ctx, tx, idempotencyKey, http.StatusOK, string(respBody)); err != nil {
		return nil, err
	}

	// Commit
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return wallet, nil
}

// Payment handles spending money from a wallet
func (s *WalletService) Payment(ctx context.Context, walletID uuid.UUID, amount decimal.Decimal, idempotencyKey string) (*models.Wallet, error) {
	shardIdx := s.getShard(walletID)
	s.mu[shardIdx].Lock()
	defer s.mu[shardIdx].Unlock()

	roundedAmount := amount.Round(2)
	if roundedAmount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("payment amount must be positive")
	}

	// Handle very small payment rejection as per requirement
	if roundedAmount.LessThan(decimal.NewFromFloat(0.01)) {
		return nil, errors.New("payment amount below minimum unit (0.01)")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Idempotency
	cachedCode, cachedBody, found, err := s.iRepo.GetResponse(ctx, idempotencyKey)
	if err == nil && found {
		if cachedCode >= 200 && cachedCode < 300 {
			var cachedWallet models.Wallet
			if err := json.Unmarshal([]byte(cachedBody), &cachedWallet); err == nil {
				return &cachedWallet, nil
			}
		}
		return nil, fmt.Errorf("duplicate request")
	}

	isDup, err := s.iRepo.CheckAndCreateKey(ctx, tx, idempotencyKey)
	if err != nil {
		return nil, err
	}
	if isDup {
		return nil, fmt.Errorf("duplicate request")
	}

	// Wallet lock
	wallet, err := s.wRepo.GetWalletByIDWithLock(ctx, tx, walletID)
	if err != nil {
		return nil, err
	}

	if wallet.Status == models.StatusSuspended {
		return nil, errors.New("wallet is suspended")
	}

	// Check Insufficient Funds
	if wallet.Balance.LessThan(roundedAmount) {
		return nil, errors.New("insufficient funds")
	}

	// Update Balance
	newBalance := wallet.Balance.Sub(roundedAmount)
	if err := s.wRepo.UpdateWalletBalance(ctx, tx, walletID, newBalance); err != nil {
		return nil, err
	}

	// Ledger
	entry := &models.LedgerEntry{
		ID:           uuid.New(),
		WalletID:     walletID,
		Amount:       roundedAmount.Neg(), // Ledger records payment as negative change
		Type:         models.TypePayment,
		BalanceAfter: newBalance,
		Description:  fmt.Sprintf("Payment of %s %s", roundedAmount, wallet.Currency),
		CreatedAt:    time.Now(),
	}
	if err := s.lRepo.CreateLedgerEntry(ctx, tx, entry); err != nil {
		return nil, err
	}

	// Save Response for Idempotency
	wallet.Balance = newBalance
	respBody, _ := json.Marshal(wallet)
	if err := s.iRepo.SaveResponse(ctx, tx, idempotencyKey, http.StatusOK, string(respBody)); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return wallet, nil
}

// Transfer handles moving money between wallets of the SAME currency
func (s *WalletService) Transfer(ctx context.Context, fromID, toID uuid.UUID, amount decimal.Decimal, idempotencyKey string) error {
	idxFrom := s.getShard(fromID)
	idxTo := s.getShard(toID)

	// Consistent locking order to prevent deadlocks
	if idxFrom == idxTo {
		s.mu[idxFrom].Lock()
		defer s.mu[idxFrom].Unlock()
	} else if idxFrom < idxTo {
		s.mu[idxFrom].Lock()
		s.mu[idxTo].Lock()
		defer s.mu[idxTo].Unlock()
		defer s.mu[idxFrom].Unlock()
	} else {
		s.mu[idxTo].Lock()
		s.mu[idxFrom].Lock()
		defer s.mu[idxFrom].Unlock()
		defer s.mu[idxTo].Unlock()
	}

	roundedAmount := amount.Round(2)
	if roundedAmount.LessThanOrEqual(decimal.Zero) {
		return errors.New("transfer amount must be positive")
	}

	if fromID == toID {
		return errors.New("cannot transfer to the same wallet")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Idempotency
	cachedCode, _, found, err := s.iRepo.GetResponse(ctx, idempotencyKey)
	if err == nil && found {
		if cachedCode >= 200 && cachedCode < 300 {
			return nil // Already successful
		}
		return fmt.Errorf("duplicate request")
	}

	isDup, err := s.iRepo.CheckAndCreateKey(ctx, tx, idempotencyKey)
	if err != nil {
		return err
	}
	if isDup {
		return fmt.Errorf("duplicate request")
	}

	// Lock both wallets. To prevent deadlocks, always lock the one with smaller UUID first.
	var wFrom, wTo *models.Wallet
	if fromID.String() < toID.String() {
		wFrom, err = s.wRepo.GetWalletByIDWithLock(ctx, tx, fromID)
		if err != nil {
			return err
		}
		wTo, err = s.wRepo.GetWalletByIDWithLock(ctx, tx, toID)
		if err != nil {
			return err
		}
	} else {
		wTo, err = s.wRepo.GetWalletByIDWithLock(ctx, tx, toID)
		if err != nil {
			return err
		}
		wFrom, err = s.wRepo.GetWalletByIDWithLock(ctx, tx, fromID)
		if err != nil {
			return err
		}
	}

	// Currency Mismatch Check
	if wFrom.Currency != wTo.Currency {
		return fmt.Errorf("currency mismatch: %s != %s", wFrom.Currency, wTo.Currency)
	}

	// Status checks
	if wFrom.Status == models.StatusSuspended || wTo.Status == models.StatusSuspended {
		return errors.New("one or both wallets are suspended")
	}

	// Balance check
	if wFrom.Balance.LessThan(roundedAmount) {
		return errors.New("insufficient funds for transfer")
	}

	// Execute Balance Updates
	newBalanceFrom := wFrom.Balance.Sub(roundedAmount)
	newBalanceTo := wTo.Balance.Add(roundedAmount)

	if err := s.wRepo.UpdateWalletBalance(ctx, tx, fromID, newBalanceFrom); err != nil {
		return err
	}
	if err := s.wRepo.UpdateWalletBalance(ctx, tx, toID, newBalanceTo); err != nil {
		return err
	}

	// Create Ledger Entries
	refID := uuid.New()
	
	entryFrom := &models.LedgerEntry{
		ID:           uuid.New(),
		WalletID:     fromID,
		Amount:       roundedAmount.Neg(),
		Type:         models.TypeTransferOut,
		BalanceAfter: newBalanceFrom,
		ReferenceID:  &refID,
		Description:  fmt.Sprintf("Transfer out to wallet %s", toID),
		CreatedAt:    time.Now(),
	}
	if err := s.lRepo.CreateLedgerEntry(ctx, tx, entryFrom); err != nil {
		return err
	}

	entryTo := &models.LedgerEntry{
		ID:           uuid.New(),
		WalletID:     toID,
		Amount:       roundedAmount,
		Type:         models.TypeTransferIn,
		BalanceAfter: newBalanceTo,
		ReferenceID:  &refID,
		Description:  fmt.Sprintf("Transfer in from wallet %s", fromID),
		CreatedAt:    time.Now(),
	}
	if err := s.lRepo.CreateLedgerEntry(ctx, tx, entryTo); err != nil {
		return err
	}

	// Save Response for Idempotency
	if err := s.iRepo.SaveResponse(ctx, tx, idempotencyKey, http.StatusOK, "OK"); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *WalletService) GetWallet(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
	return s.wRepo.GetWalletByID(ctx, id)
}

func (s *WalletService) SuspendWallet(ctx context.Context, id uuid.UUID) error {
	shardIdx := s.getShard(id)
	s.mu[shardIdx].Lock()
	defer s.mu[shardIdx].Unlock()

	return s.wRepo.UpdateWalletStatus(ctx, id, models.StatusSuspended)
}
