// Package dto holds Core HTTP request/response shapes, kept separate from
// domain entities so wire format changes never leak into business logic.
package dto

// RegisterRequest is the POST /auth/register body.
type RegisterRequest struct {
	Name                string `json:"name"`
	Email               string `json:"email"`
	Password            string `json:"password"`
	PasswordConfirmation string `json:"password_confirmation"`
}

// LoginRequest is the POST /auth/login body.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RefreshRequest is the POST /auth/refresh body.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// UserResponse is the public shape of a user returned from auth endpoints.
type UserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// AuthResponse is returned by /auth/login and /auth/refresh.
type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"`
	TokenType    string       `json:"token_type"`
	User         UserResponse `json:"user"`
}
