package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/XoDeR/empops/api-go/internal/domain/entity"
	"github.com/XoDeR/empops/api-go/internal/domain/repository"
)

type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (*entity.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, email, name, password_hash, created_at, updated_at
		FROM users WHERE id = $1`, id)

	var u entity.User
	if err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, email, name, password_hash, created_at, updated_at
		FROM users WHERE email = $1`, email)

	var u entity.User
	if err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *PostgresUserRepository) Create(ctx context.Context, u *entity.User) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (id, email, name, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, now(), now())`,
		u.ID, u.Email, u.Name, u.PasswordHash,
	)
	return err
}

type PostgresRefreshTokenRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRefreshTokenRepository(pool *pgxpool.Pool) *PostgresRefreshTokenRepository {
	return &PostgresRefreshTokenRepository{pool: pool}
}

func (r *PostgresRefreshTokenRepository) Create(ctx context.Context, id, userID, jti string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (id, user_id, jti, expires_at, created_at)
		VALUES ($1, $2, $3, $4, now())`, id, userID, jti, expiresAt)
	return err
}

func (r *PostgresRefreshTokenRepository) FindByJTI(ctx context.Context, jti string) (string, time.Time, *time.Time, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT user_id, expires_at, revoked_at FROM refresh_tokens WHERE jti = $1`, jti)

	var userID string
	var expiresAt time.Time
	var revokedAt *time.Time
	if err := row.Scan(&userID, &expiresAt, &revokedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", time.Time{}, nil, repository.ErrRefreshNotFound
		}
		return "", time.Time{}, nil, err
	}
	return userID, expiresAt, revokedAt, nil
}

func (r *PostgresRefreshTokenRepository) RevokeByJTI(ctx context.Context, jti string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = now() WHERE jti = $1 AND revoked_at IS NULL`, jti)
	return err
}
