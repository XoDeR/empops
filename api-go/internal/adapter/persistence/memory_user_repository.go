// Package persistence contains Core repository implementations. Step 0
// ships an in-memory stub so auth works without a database; a SQLC/pgx
// backed implementation will replace it once migrations/core is wired up.
package persistence

import (
	"context"
	"sync"

	"github.com/XoDeR/empops/api-go/internal/domain/entity"
	"github.com/XoDeR/empops/api-go/internal/domain/repository"
)

// MemoryUserRepository is a thread-safe in-memory repository.UserRepository.
type MemoryUserRepository struct {
	mu    sync.RWMutex
	byID  map[string]*entity.User
	email map[string]string // email -> id
}

// NewMemoryUserRepository returns an empty in-memory user repository.
func NewMemoryUserRepository() *MemoryUserRepository {
	return &MemoryUserRepository{
		byID:  make(map[string]*entity.User),
		email: make(map[string]string),
	}
}

func (r *MemoryUserRepository) FindByID(_ context.Context, id string) (*entity.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	u, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	copy := *u
	return &copy, nil
}

func (r *MemoryUserRepository) FindByEmail(_ context.Context, email string) (*entity.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.email[email]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	copy := *r.byID[id]
	return &copy, nil
}

func (r *MemoryUserRepository) Create(_ context.Context, u *entity.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	copy := *u
	r.byID[u.ID] = &copy
	r.email[u.Email] = u.ID
	return nil
}

var _ repository.UserRepository = (*MemoryUserRepository)(nil)
