// Package usecase implements Core application flows (auth, RBAC, me),
// orchestrating domain entities and repository ports.
package usecase

import (
	"context"
	"errors"

	"github.com/XoDeR/empops/api-go/internal/domain/entity"
	"github.com/XoDeR/empops/api-go/internal/domain/repository"
	"github.com/XoDeR/empops/api-go/pkg/jwt"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

// ErrInvalidCredentials is returned when login fails (Step 0 stub never
// actually returns this, since any email/password is accepted).
var ErrInvalidCredentials = errors.New("usecase: invalid credentials")

// AuthResult is the outcome of a successful login or refresh.
type AuthResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	User         *entity.User
}

// AuthUseCase implements Core authentication flows.
//
// Step 0 note: this is a STUB. It accepts any email/password, lazily
// creating (or reusing) a matching in-memory user so the frontend can
// integrate against a real JWT flow before Postgres-backed auth lands.
type AuthUseCase struct {
	users repository.UserRepository
	jwt   *jwt.Manager
}

// NewAuthUseCase builds an AuthUseCase from its dependencies.
func NewAuthUseCase(users repository.UserRepository, jwtManager *jwt.Manager) *AuthUseCase {
	return &AuthUseCase{users: users, jwt: jwtManager}
}

// Login accepts any non-empty email/password (Step 0 stub) and returns a
// fresh token pair for a stub user matching that email.
func (uc *AuthUseCase) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	if email == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := uc.users.FindByEmail(ctx, email)
	if errors.Is(err, repository.ErrUserNotFound) {
		user, err = entity.NewUser(uuidv7.New(), email, stubNameFromEmail(email), "")
		if err != nil {
			return nil, err
		}
		if err := uc.users.Create(ctx, user); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	pair, err := uc.jwt.IssuePair(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
		User:         user,
	}, nil
}

// Refresh validates a refresh token and issues a new access token.
func (uc *AuthUseCase) Refresh(ctx context.Context, refreshToken string) (*AuthResult, error) {
	claims, err := uc.jwt.Parse(refreshToken, jwt.TokenTypeRefresh)
	if err != nil {
		return nil, err
	}

	user, err := uc.users.FindByID(ctx, claims.Subject)
	if err != nil {
		return nil, err
	}

	accessToken, expiresIn, err := uc.jwt.IssueAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		User:         user,
	}, nil
}

// Me returns the user identified by a validated access token subject.
func (uc *AuthUseCase) Me(ctx context.Context, userID string) (*entity.User, error) {
	return uc.users.FindByID(ctx, userID)
}

func stubNameFromEmail(email string) string {
	for i, c := range email {
		if c == '@' {
			return email[:i]
		}
	}
	return email
}
