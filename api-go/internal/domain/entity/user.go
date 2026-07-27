// Package entity holds Core domain entities and their invariants. Entities
// depend only on the standard library and pkg value types, never on
// adapters or infrastructure.
package entity

import (
	"errors"
	"time"
)

// ErrInvalidEmail is returned when a User is constructed with an empty email.
var ErrInvalidEmail = errors.New("entity: email must not be empty")

// User is the Core account entity backing authentication and RBAC.
type User struct {
	ID           string
	Email        string
	Name         string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewUser builds a User, validating required invariants.
func NewUser(id, email, name, passwordHash string) (*User, error) {
	if email == "" {
		return nil, ErrInvalidEmail
	}
	now := time.Now().UTC()
	return &User{
		ID:           id,
		Email:        email,
		Name:         name,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}
