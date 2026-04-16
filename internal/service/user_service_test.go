package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"voca-test/internal/repository"
)

func TestUserService_CreateUser(t *testing.T) {
	mockRepo := new(repository.MockUserRepository)
	service := NewUserService(mockRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		name := "Test User"
		mockRepo.On("CreateUser", ctx, mock.Anything).Return(nil).Once()

		user, err := service.CreateUser(ctx, name)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, name, user.Name)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Error", func(t *testing.T) {
		name := "Error User"
		mockRepo.On("CreateUser", ctx, mock.Anything).Return(errors.New("db error")).Once()

		user, err := service.CreateUser(ctx, name)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "db error")
		mockRepo.AssertExpectations(t)
	})
}
