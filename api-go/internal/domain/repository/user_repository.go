// Package repository declares Core repository ports (interfaces only).
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/XoDeR/empops/api-go/internal/domain/entity"
)

var (
	ErrUserNotFound      = errors.New("repository: user not found")
	ErrRefreshNotFound   = errors.New("repository: refresh token not found")
	ErrRefreshRevoked    = errors.New("repository: refresh token revoked")
)

// UserRepository is the persistence port for the User entity.
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	Create(ctx context.Context, u *entity.User) error
}

// RefreshTokenRepository stores refresh-token JTIs for rotation/revocation.
type RefreshTokenRepository interface {
	Create(ctx context.Context, id, userID, jti string, expiresAt time.Time) error
	FindByJTI(ctx context.Context, jti string) (userID string, expiresAt time.Time, revokedAt *time.Time, err error)
	RevokeByJTI(ctx context.Context, jti string) error
}
