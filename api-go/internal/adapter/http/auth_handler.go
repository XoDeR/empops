package httpadapter

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/XoDeR/empops/api-go/internal/adapter/http/dto"
	"github.com/XoDeR/empops/api-go/internal/adapter/http/middleware"
	"github.com/XoDeR/empops/api-go/internal/domain/entity"
	"github.com/XoDeR/empops/api-go/internal/usecase"
	"github.com/XoDeR/empops/api-go/pkg/response"
)

// AuthHandler exposes the Core authentication endpoints backed by AuthUseCase.
type AuthHandler struct {
	authUseCase *usecase.AuthUseCase
}

// NewAuthHandler builds an AuthHandler from its dependencies.
func NewAuthHandler(authUseCase *usecase.AuthUseCase) *AuthHandler {
	return &AuthHandler{authUseCase: authUseCase}
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	result, err := h.authUseCase.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidCredentials) {
			response.Fail(w, http.StatusUnauthorized, "invalid credentials", nil)
			return
		}
		response.Fail(w, http.StatusInternalServerError, "login failed", err.Error())
		return
	}

	response.OK(w, "logged in", authResultToResponse(result))
}

// Refresh handles POST /auth/refresh.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	result, err := h.authUseCase.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		response.Fail(w, http.StatusUnauthorized, "invalid or expired refresh token", nil)
		return
	}

	response.OK(w, "token refreshed", authResultToResponse(result))
}

// Logout handles POST /auth/logout. Step 0 has no server-side token
// revocation store yet, so this is a no-op that always succeeds; the client
// is expected to discard its tokens.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	response.OK(w, "logged out", nil)
}

// Me handles GET /auth/me, requiring middleware.RequireAuth to have run.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusUnauthorized, "missing authenticated user", nil)
		return
	}

	user, err := h.authUseCase.Me(r.Context(), userID)
	if err != nil {
		response.Fail(w, http.StatusNotFound, "user not found", nil)
		return
	}

	response.OK(w, "ok", userToResponse(user))
}

func authResultToResponse(result *usecase.AuthResult) dto.AuthResponse {
	return dto.AuthResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
		TokenType:    "Bearer",
		User:         userToResponse(result.User),
	}
}

func userToResponse(u *entity.User) dto.UserResponse {
	return dto.UserResponse{
		ID:    u.ID,
		Email: u.Email,
		Name:  u.Name,
	}
}
