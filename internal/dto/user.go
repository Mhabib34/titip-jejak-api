package dto

import (
	"titip-jejak-api/internal/model"
	"time"

	"github.com/google/uuid"
)

type RegisterRequest struct {
	Name     string     `json:"name"     binding:"required"`
	Email    string     `json:"email"    binding:"required,email"`
	Password string     `json:"password" binding:"required"`
	Role     model.Role `json:"role"     binding:"required,oneof=finder seeker volunteer"`
	Phone    string     `json:"phone"`
}

type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// ── Response DTOs ─────────────────────────────────────────────────────────────

type UserResponse struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Role      model.Role `json:"role"`
	Phone     *string    `json:"phone"`
	CreatedAt time.Time  `json:"created_at"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type LoginResponse struct {
	User   *UserResponse `json:"user"`
	Tokens *TokenPair    `json:"tokens,omitempty"` // nil untuk web (token di cookie)
}

type RefreshResponse struct {
	AccessToken string `json:"access_token,omitempty"` // nil untuk web (token di cookie)
}
