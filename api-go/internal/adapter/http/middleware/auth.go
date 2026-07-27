// Package middleware holds Core HTTP middleware (auth, etc.) shared across
// routes; modules receive equivalent helpers via pkg or module.Core rather
// than importing this package directly.
package middleware

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
// context on success.
func RequireAuth(manager *jwt.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token, ok := bearerToken(header)
			if !ok {
				response.Fail(w, http.StatusUnauthorized, "missing or invalid Authorization header", nil)
				return
			}

			claims, err := manager.Parse(token, jwt.TokenTypeAccess)
			if err != nil {
				response.Fail(w, http.StatusUnauthorized, "invalid or expired access token", nil)
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
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", false
	}
	return token, true
}
