// Package repository declares Core repository ports (interfaces only).
// Concrete implementations live in internal/adapter/persistence.
package repository

import (
	"context"
	"errors"

	"github.com/XoDeR/empops/api-go/internal/domain/entity"
)

// ErrUserNotFound is returned when no user matches the lookup criteria.
var ErrUserNotFound = errors.New("repository: user not found")

// UserRepository is the persistence port for the User entity.
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	Create(ctx context.Context, u *entity.User) error
}
