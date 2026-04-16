package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"voca-test/internal/models"
)

// LedgerRepository handles ledger data operations
type LedgerRepository interface {
	CreateLedgerEntry(ctx context.Context, tx pgx.Tx, entry *models.LedgerEntry) error
	GetLedgerByWalletID(ctx context.Context, walletID uuid.UUID) ([]models.LedgerEntry, error)
}

type ledgerRepo struct {
	db *pgxpool.Pool
}

func NewLedgerRepository(db *pgxpool.Pool) LedgerRepository {
	return &ledgerRepo{db: db}
}

func (r *ledgerRepo) CreateLedgerEntry(ctx context.Context, tx pgx.Tx, entry *models.LedgerEntry) error {
	query := `INSERT INTO ledger (id, wallet_id, amount, type, balance_after, reference_id, description, created_at) 
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := tx.Exec(ctx, query, entry.ID, entry.WalletID, entry.Amount, entry.Type, entry.BalanceAfter, entry.ReferenceID, entry.Description, entry.CreatedAt)
	if err != nil {
		return fmt.Errorf("error creating ledger entry: %w", err)
	}
	return nil
}

func (r *ledgerRepo) GetLedgerByWalletID(ctx context.Context, walletID uuid.UUID) ([]models.LedgerEntry, error) {
	query := `SELECT id, wallet_id, amount, type, balance_after, reference_id, description, created_at 
              FROM ledger 
              WHERE wallet_id = $1 
              ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query, walletID)
	if err != nil {
		return nil, fmt.Errorf("error fetching ledger history: %w", err)
	}
	defer rows.Close()

	var entries []models.LedgerEntry
	for rows.Next() {
		var entry models.LedgerEntry
		err := rows.Scan(&entry.ID, &entry.WalletID, &entry.Amount, &entry.Type, &entry.BalanceAfter, &entry.ReferenceID, &entry.Description, &entry.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("error scanning ledger row: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}
