package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"voca-test/internal/models"
)

// WalletRepository handles wallet data operations
type WalletRepository interface {
	CreateWallet(ctx context.Context, wallet *models.Wallet) error
	GetWalletByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error)
	GetWalletByIDWithLock(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*models.Wallet, error)
	GetWalletByUserAndCurrency(ctx context.Context, userID uuid.UUID, currency string) (*models.Wallet, error)
	UpdateWalletBalance(ctx context.Context, tx pgx.Tx, walletID uuid.UUID, newBalance interface{}) error
	UpdateWalletStatus(ctx context.Context, walletID uuid.UUID, status string) error
}

type walletRepo struct {
	db *pgxpool.Pool
}

func NewWalletRepository(db *pgxpool.Pool) WalletRepository {
	return &walletRepo{db: db}
}

func (r *walletRepo) CreateWallet(ctx context.Context, wallet *models.Wallet) error {
	query := `INSERT INTO wallets (id, user_id, currency, balance, status, created_at, updated_at) 
              VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.Exec(ctx, query, wallet.ID, wallet.UserID, wallet.Currency, wallet.Balance, wallet.Status, wallet.CreatedAt, wallet.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error creating wallet: %w", err)
	}
	return nil
}

func (r *walletRepo) GetWalletByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
	var wallet models.Wallet
	query := `SELECT id, user_id, currency, balance, status, created_at, updated_at FROM wallets WHERE id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(&wallet.ID, &wallet.UserID, &wallet.Currency, &wallet.Balance, &wallet.Status, &wallet.CreatedAt, &wallet.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("wallet not found")
		}
		return nil, fmt.Errorf("error fetching wallet: %w", err)
	}
	return &wallet, nil
}

func (r *walletRepo) GetWalletByIDWithLock(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*models.Wallet, error) {
	var wallet models.Wallet
	query := `SELECT id, user_id, currency, balance, status, created_at, updated_at 
              FROM wallets 
              WHERE id = $1 
              FOR UPDATE`
	err := tx.QueryRow(ctx, query, id).Scan(&wallet.ID, &wallet.UserID, &wallet.Currency, &wallet.Balance, &wallet.Status, &wallet.CreatedAt, &wallet.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("wallet not found for update")
		}
		return nil, fmt.Errorf("error fetching wallet with lock: %w", err)
	}
	return &wallet, nil
}

func (r *walletRepo) GetWalletByUserAndCurrency(ctx context.Context, userID uuid.UUID, currency string) (*models.Wallet, error) {
	var wallet models.Wallet
	query := `SELECT id, user_id, currency, balance, status, created_at, updated_at 
              FROM wallets 
              WHERE user_id = $1 AND currency = $2`
	err := r.db.QueryRow(ctx, query, userID, currency).Scan(&wallet.ID, &wallet.UserID, &wallet.Currency, &wallet.Balance, &wallet.Status, &wallet.CreatedAt, &wallet.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error fetching wallet by user and currency: %w", err)
	}
	return &wallet, nil
}

func (r *walletRepo) UpdateWalletBalance(ctx context.Context, tx pgx.Tx, id uuid.UUID, balance interface{}) error {
	query := `UPDATE wallets SET balance = $1 WHERE id = $2`
	_, err := tx.Exec(ctx, query, balance, id)
	return err
}

func (r *walletRepo) UpdateWalletStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE wallets SET status = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, status, id)
	return err
}
