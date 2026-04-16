package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"voca-test/internal/models"
	"voca-test/internal/repository"
)

type UserService struct {
	userRepo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{
		userRepo: repo,
	}
}

// CreateUser handles creating a new user
func (s *UserService) CreateUser(ctx context.Context, name string) (*models.User, error) {
	user := &models.User{
		ID:        uuid.New(),
		Name:      name,
		CreatedAt: time.Now(),
	}
	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}
