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

// UserRepository handles user data operations
type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

type userRepo struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) CreateUser(ctx context.Context, user *models.User) error {
	query := `INSERT INTO users (id, name, created_at) VALUES ($1, $2, $3)`
	_, err := r.db.Exec(ctx, query, user.ID, user.Name, user.CreatedAt)
	if err != nil {
		return fmt.Errorf("error creating user: %w", err)
	}
	return nil
}

func (r *userRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	query := `SELECT id, name, created_at FROM users WHERE id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(&user.ID, &user.Name, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("error fetching user: %w", err)
	}
	return &user, nil
}
