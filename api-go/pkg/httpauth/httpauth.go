// Package httpauth provides the shared JWT Bearer-auth HTTP middleware used
// by Core and every vertical module. It lives in pkg (not internal/adapter)
// specifically so modules can require authentication without importing
// Core's internal packages (see architecture-go-ddd.md anti-pattern #2).
package httpauth

import (
	"context"
	"net/http"
	"strings"

	"github.com/XoDeR/empops/api-go/pkg/jwt"
	"github.com/XoDeR/empops/api-go/pkg/response"
)

type contextKey string

// UserIDContextKey is the context key holding the authenticated user's ID,
// set by RequireAuth after a valid Bearer access token is parsed.
const UserIDContextKey contextKey = "userID"

// RequireAuth returns middleware that rejects requests without a valid
// Bearer access token, storing the token subject (user ID) in the request
// context on success. Mirrors Laravel's AuthenticateJwt middleware.
func RequireAuth(manager *jwt.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token, ok := bearerToken(header)
			if !ok {
				response.Fail(w, http.StatusUnauthorized, "Missing or invalid Authorization header", nil)
				return
			}

			claims, err := manager.Parse(token, jwt.TokenTypeAccess)
			if err != nil {
				response.Fail(w, http.StatusUnauthorized, "Invalid or expired token", nil)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDContextKey, claims.Subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext extracts the authenticated user ID set by RequireAuth.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(UserIDContextKey).(string)
	return id, ok
}

func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) && !strings.HasPrefix(header, "bearer ") {
		return "", false
	}
	token := strings.TrimSpace(header[len(prefix):])
	if token == "" {
		return "", false
	}
	return token, true
}
