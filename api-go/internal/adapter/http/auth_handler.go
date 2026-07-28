package httpadapter

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/XoDeR/empops/api-go/internal/adapter/http/dto"
	"github.com/XoDeR/empops/api-go/pkg/httpauth"
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

// Register handles POST /auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	result, err := h.authUseCase.Register(r.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, usecase.ErrEmailTaken) {
			response.Fail(w, http.StatusUnprocessableEntity, "email already registered", nil)
			return
		}
		response.Fail(w, http.StatusInternalServerError, "registration failed", err.Error())
		return
	}

	response.Created(w, "Registered", authResultToResponse(result))
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

// Logout handles POST /auth/logout and revokes the refresh token when provided.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	_ = h.authUseCase.Logout(r.Context(), req.RefreshToken)
	response.OK(w, "logged out", nil)
}

// Me handles GET /auth/me, requiring middleware.RequireAuth to have run.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpauth.UserIDFromContext(r.Context())
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
