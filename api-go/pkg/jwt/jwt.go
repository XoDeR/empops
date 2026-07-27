// Package jwt wraps golang-jwt/jwt/v5 with the access/refresh token pair
// used by EmpOps Core auth, matching the Laravel firebase/php-jwt setup
// (HS256, sub/jti/iat/exp/iss/aud claims).
package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenType distinguishes access vs refresh tokens in the claims.
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

// ErrInvalidToken is returned for any token that fails parsing/validation.
var ErrInvalidToken = errors.New("jwt: invalid token")

// Config holds the settings needed to sign and verify tokens.
type Config struct {
	Secret          string
	Issuer          string
	Audience        string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

// Claims is the custom claim set embedded in every EmpOps token.
type Claims struct {
	Type TokenType `json:"type"`
	jwt.RegisteredClaims
}

// Manager issues and verifies access/refresh token pairs.
type Manager struct {
	cfg Config
}

// NewManager builds a Manager from cfg.
func NewManager(cfg Config) *Manager {
	return &Manager{cfg: cfg}
}

// TokenPair is the result of issuing a new access + refresh token pair.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64 // seconds until the access token expires
}

// IssuePair creates a new access token and refresh token for the given subject (user ID).
func (m *Manager) IssuePair(subject string) (TokenPair, error) {
	access, err := m.issue(subject, TokenTypeAccess, m.cfg.AccessTokenTTL)
	if err != nil {
		return TokenPair{}, err
	}

	refresh, err := m.issue(subject, TokenTypeRefresh, m.cfg.RefreshTokenTTL)
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(m.cfg.AccessTokenTTL.Seconds()),
	}, nil
}

// IssueAccessToken creates a standalone access token, used when refreshing.
func (m *Manager) IssueAccessToken(subject string) (string, int64, error) {
	token, err := m.issue(subject, TokenTypeAccess, m.cfg.AccessTokenTTL)
	if err != nil {
		return "", 0, err
	}
	return token, int64(m.cfg.AccessTokenTTL.Seconds()), nil
}

func (m *Manager) issue(subject string, tokenType TokenType, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		Type: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Issuer:    m.cfg.Issuer,
			Audience:  jwt.ClaimStrings{m.cfg.Audience},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.cfg.Secret))
}

// Parse validates raw and returns its claims. It also enforces the expected
// token type when expected is non-empty.
func (m *Manager) Parse(raw string, expected TokenType) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(m.cfg.Secret), nil
	},
		jwt.WithIssuer(m.cfg.Issuer),
		jwt.WithAudience(m.cfg.Audience),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
	)
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	if expected != "" && claims.Type != expected {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
