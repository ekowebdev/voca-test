package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IdempotencyRepository handles idempotency key operations
type IdempotencyRepository interface {
	CheckAndCreateKey(ctx context.Context, tx pgx.Tx, key string) (bool, error)
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
