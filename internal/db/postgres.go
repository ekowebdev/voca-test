package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"voca-test/internal/util"
)

// DB instance
type DB struct {
	Pool *pgxpool.Pool
}

// ConnectPostgres establishes a connection to the PostgreSQL database
func ConnectPostgres(cfg *util.Config) (*DB, error) {
	// 1. Create pool config
	poolConfig, err := pgxpool.ParseConfig(cfg.DBConn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection string: %v", err)
	}

	// 2. Apply pool settings
	poolConfig.MaxConns = cfg.DBPool.MaxConns
	poolConfig.MinConns = cfg.DBPool.MinConns
	poolConfig.MaxConnLifetime = time.Duration(cfg.DBPool.MaxConnLifetime) * time.Minute
	poolConfig.MaxConnIdleTime = time.Duration(cfg.DBPool.MaxConnIdleTime) * time.Minute

	// 3. Create a connection pool
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}

	// 4. Verify connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to ping database: %v", err)
	}

	return &DB{Pool: pool}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	db.Pool.Close()
}
