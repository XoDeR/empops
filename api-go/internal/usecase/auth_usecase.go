package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/XoDeR/empops/api-go/internal/domain/entity"
	"github.com/XoDeR/empops/api-go/internal/domain/repository"
	"github.com/XoDeR/empops/api-go/pkg/jwt"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

var (
	ErrInvalidCredentials = errors.New("usecase: invalid credentials")
	ErrEmailTaken         = errors.New("usecase: email already registered")
	ErrRefreshInvalid     = errors.New("usecase: invalid or expired refresh token")
)

type AuthResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	User         *entity.User
}

type AuthUseCase struct {
	users    repository.UserRepository
	refresh  repository.RefreshTokenRepository
	jwt      *jwt.Manager
}

func NewAuthUseCase(users repository.UserRepository, refresh repository.RefreshTokenRepository, jwtManager *jwt.Manager) *AuthUseCase {
	return &AuthUseCase{users: users, refresh: refresh, jwt: jwtManager}
}

func (uc *AuthUseCase) Register(ctx context.Context, name, email, password string) (*AuthResult, error) {
	if email == "" || password == "" || name == "" {
		return nil, ErrInvalidCredentials
	}

	if _, err := uc.users.FindByEmail(ctx, email); err == nil {
		return nil, ErrEmailTaken
	} else if !errors.Is(err, repository.ErrUserNotFound) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user, err := entity.NewUser(uuidv7.New(), email, name, string(hash))
	if err != nil {
		return nil, err
	}
	if err := uc.users.Create(ctx, user); err != nil {
		return nil, err
	}

	return uc.issueAuthResult(ctx, user)
}

func (uc *AuthUseCase) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	if email == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := uc.users.FindByEmail(ctx, email)
	if errors.Is(err, repository.ErrUserNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return uc.issueAuthResult(ctx, user)
}

func (uc *AuthUseCase) Refresh(ctx context.Context, refreshToken string) (*AuthResult, error) {
	claims, err := uc.jwt.Parse(refreshToken, jwt.TokenTypeRefresh)
	if err != nil {
		return nil, ErrRefreshInvalid
	}

	jti := claims.ID
	userID := claims.Subject

	storedUserID, expiresAt, revokedAt, err := uc.refresh.FindByJTI(ctx, jti)
	if errors.Is(err, repository.ErrRefreshNotFound) {
		return nil, ErrRefreshInvalid
	}
	if err != nil {
		return nil, err
	}
	if storedUserID != userID || revokedAt != nil || time.Now().After(expiresAt) {
		return nil, ErrRefreshInvalid
	}

	if err := uc.refresh.RevokeByJTI(ctx, jti); err != nil {
		return nil, err
	}

	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return nil, ErrRefreshInvalid
	}

	return uc.issueAuthResult(ctx, user)
}

func (uc *AuthUseCase) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	claims, err := uc.jwt.Parse(refreshToken, jwt.TokenTypeRefresh)
	if err != nil {
		return nil
	}
	return uc.refresh.RevokeByJTI(ctx, claims.ID)
}

func (uc *AuthUseCase) Me(ctx context.Context, userID string) (*entity.User, error) {
	return uc.users.FindByID(ctx, userID)
}

func (uc *AuthUseCase) issueAuthResult(ctx context.Context, user *entity.User) (*AuthResult, error) {
	pair, err := uc.jwt.IssuePair(user.ID)
	if err != nil {
		return nil, err
	}

	if err := uc.refresh.Create(ctx, uuid.NewString(), user.ID, pair.RefreshJTI, pair.RefreshExpiresAt); err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
		User:         user,
	}, nil
}
