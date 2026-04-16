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
	GetLedgerByWalletID(ctx context.Context, walletID uuid.UUID, txType string, limit, offset int) ([]models.LedgerEntry, error)
	CountLedgerByWalletID(ctx context.Context, walletID uuid.UUID, txType string) (int64, error)
	GetLedgerSummary(ctx context.Context, walletID uuid.UUID, txType string) (map[string]interface{}, error)
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

func (r *ledgerRepo) GetLedgerByWalletID(ctx context.Context, walletID uuid.UUID, txType string, limit, offset int) ([]models.LedgerEntry, error) {
	query := `SELECT id, wallet_id, amount, type, balance_after, reference_id, description, created_at 
              FROM ledger 
              WHERE wallet_id = $1`

	args := []interface{}{walletID}
	nextArg := 2
	if txType != "" {
		query += fmt.Sprintf(" AND type = $%d", nextArg)
		args = append(args, txType)
		nextArg++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", nextArg, nextArg+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
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

func (r *ledgerRepo) CountLedgerByWalletID(ctx context.Context, walletID uuid.UUID, txType string) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM ledger WHERE wallet_id = $1`

	args := []interface{}{walletID}
	if txType != "" {
		query += " AND type = $2"
		args = append(args, txType)
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error counting ledger entries: %w", err)
	}
	return count, nil
}

func (r *ledgerRepo) GetLedgerSummary(ctx context.Context, walletID uuid.UUID, txType string) (map[string]interface{}, error) {
	query := `SELECT 
                COALESCE(SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END), 0) as total_credit,
                COALESCE(SUM(CASE WHEN amount < 0 THEN amount ELSE 0 END), 0) as total_debit
              FROM ledger 
              WHERE wallet_id = $1`

	args := []interface{}{walletID}
	if txType != "" {
		query += " AND type = $2"
		args = append(args, txType)
	}

	var totalCredit, totalDebit interface{}
	err := r.db.QueryRow(ctx, query, args...).Scan(&totalCredit, &totalDebit)
	if err != nil {
		return nil, fmt.Errorf("error getting ledger summary: %w", err)
	}

	return map[string]interface{}{
		"total_credit": totalCredit,
		"total_debit":  totalDebit,
	}, nil
}
