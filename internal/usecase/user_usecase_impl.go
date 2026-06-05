package usecase

import (
	"context"
	"fmt"
	"titip-jejak-api/internal/dto"
	"titip-jejak-api/internal/exception"
	"titip-jejak-api/internal/helper"
	"titip-jejak-api/internal/model"
	"titip-jejak-api/internal/repository"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type UserUsecaseImpl struct {
	repo     repository.UserRepository
	Validate *validator.Validate
}

func NewUserUsecase(repo repository.UserRepository, validate *validator.Validate) UserUsecase {
	return &UserUsecaseImpl{
		repo:     repo,
		Validate: validate,
	}
}

func (u *UserUsecaseImpl) Create(ctx context.Context, request *dto.RegisterRequest) (*dto.UserResponse, error) {
	err := u.Validate.Struct(request)
	exception.PanicIfError(err)

	_, err = u.repo.FIndByEmail(ctx, request.Email)
	if err == nil {
		return &dto.UserResponse{}, exception.NewConflictError("Email already exists")
	}

	hasPassword, err := helper.HashPassword(request.Password)
	exception.PanicIfError(err)

	user := &model.User{
		Name:     request.Name,
		Email:    request.Email,
		Password: hasPassword,
		Role:     request.Role,
	}

	if request.Phone != "" {
		user.Phone = &request.Phone
	}

	result, err := u.repo.Create(ctx, user)
	exception.PanicIfError(err)

	return helper.ToUserResponse(*result), nil
}

func (u *UserUsecaseImpl) Login(ctx context.Context, request *dto.LoginRequest) (*dto.LoginResponse, error) {
	err := u.Validate.Struct(request)
	exception.PanicIfError(err)

	user, err := u.repo.FIndByEmail(ctx, request.Email)
	if err != nil {
		return nil, exception.NewNotFoundError("Email not found")
	}

	if !helper.CheckPasswordHash(request.Password, user.Password) {
		return nil, exception.NewUnauthorizedError("Wrong password")
	}

	payload := helper.JwtPayload{
		ID:    user.ID,
		Email: user.Email,
	}

	accessToken, err := helper.GenerateAccessToken(payload)
	exception.PanicIfError(err)

	refreshToken, err := helper.GenerateRefreshToken(payload)
	exception.PanicIfError(err)

	return &dto.LoginResponse{
		User: helper.ToUserResponse(*user),
		Tokens: &dto.TokenPair{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	}, nil
}

func (u *UserUsecaseImpl) RefreshToken(ctx context.Context, refreshToken string) (*dto.RefreshResponse, error) {
	if refreshToken == "" {
		panic(exception.NewUnauthorizedError("refresh token is required"))
	}

	claims, err := helper.VerifyRefreshToken(refreshToken)
	if err != nil {
		return nil, exception.NewUnauthorizedError("invalid or expired refresh token")
	}

	subRaw, ok := (*claims)["sub"]
	if !ok {
		return nil, exception.NewUnauthorizedError("invalid token claims")
	}

	userID, err := uuid.Parse(fmt.Sprintf("%v", subRaw))
	if err != nil {
		return nil, exception.NewUnauthorizedError("invalid token claims")
	}

	user, err := u.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, exception.NewUnauthorizedError("user not found")
	}

	payload := helper.JwtPayload{
		ID:    user.ID,
		Email: user.Email,
	}

	newAccessToken, err := helper.GenerateAccessToken(payload)
	exception.PanicIfError(err)

	return &dto.RefreshResponse{
		AccessToken: newAccessToken,
	}, nil
}

func (u *UserUsecaseImpl) Me(ctx context.Context, userID string) (*dto.UserResponse, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, exception.NewUnauthorizedError("invalid user id")
	}

	user, err := u.repo.FindByID(ctx, id)
	if err != nil {
		return nil, exception.NewNotFoundError("user not found")
	}

	return helper.ToUserResponse(*user), nil
}
