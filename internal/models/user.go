package models

import (
	"time"

	"github.com/google/uuid"
)

// UserCreateRequest is the body for creating a user
type UserCreateRequest struct {
	Name string `json:"name" binding:"required"`
}

// User represents a system user
type User struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name" binding:"required"`
	CreatedAt time.Time `json:"created_at"`
}
