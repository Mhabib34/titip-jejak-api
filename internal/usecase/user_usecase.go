package usecase

import (
	"context"
	"titip-jejak-api/internal/dto"
)

type UserUsecase interface {
	Create(ctx context.Context, request *dto.RegisterRequest) (*dto.UserResponse, error)
	Login(ctx context.Context, request *dto.LoginRequest) (*dto.LoginResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*dto.RefreshResponse, error)
	Me(ctx context.Context, userID string) (*dto.UserResponse, error)
}
