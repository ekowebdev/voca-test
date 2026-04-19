package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IdempotencyRepository handles idempotency key operations
type IdempotencyRepository interface {
	CheckAndCreateKey(ctx context.Context, tx pgx.Tx, key string) (bool, error)
	SaveResponse(ctx context.Context, tx pgx.Tx, key string, code int, body string) error
	GetResponse(ctx context.Context, key string) (int, string, bool, error)
	DeleteExpiredKeys(ctx context.Context, olderThan time.Duration) (int64, error)
}

type idempotencyRepo struct {
	db *pgxpool.Pool
}

func NewIdempotencyRepository(db *pgxpool.Pool) IdempotencyRepository {
	return &idempotencyRepo{db: db}
}

func (r *idempotencyRepo) CheckAndCreateKey(ctx context.Context, tx pgx.Tx, key string) (bool, error) {
	query := `INSERT INTO idempotency_keys (key) VALUES ($1) ON CONFLICT (key) DO NOTHING RETURNING key`
	var returnedKey string
	err := tx.QueryRow(ctx, query, key).Scan(&returnedKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return true, nil
		}
		return false, fmt.Errorf("error checking idempotency key: %w", err)
	}
	return false, nil
}

func (r *idempotencyRepo) SaveResponse(ctx context.Context, tx pgx.Tx, key string, code int, body string) error {
	query := `UPDATE idempotency_keys SET response_code = $1, response_body = $2 WHERE key = $3`
	_, err := tx.Exec(ctx, query, code, body, key)
	return err
}

func (r *idempotencyRepo) GetResponse(ctx context.Context, key string) (int, string, bool, error) {
	query := `SELECT response_code, response_body FROM idempotency_keys WHERE key = $1`
	var code *int
	var body *string
	err := r.db.QueryRow(ctx, query, key).Scan(&code, &body)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, "", false, nil
		}
		return 0, "", false, err
	}

	if code == nil {
		return 0, "", false, nil // Still processing
	}

	return *code, *body, true, nil
}

func (r *idempotencyRepo) DeleteExpiredKeys(ctx context.Context, olderThan time.Duration) (int64, error) {
	query := `DELETE FROM idempotency_keys WHERE created_at < $1`
	cutoff := time.Now().Add(-olderThan)
	res, err := r.db.Exec(ctx, query, cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected(), nil
}
