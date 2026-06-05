package handler

import (
	"net/http"
	"titip-jejak-api/internal/dto"
	"titip-jejak-api/internal/exception"
	"titip-jejak-api/internal/helper"
	"titip-jejak-api/internal/usecase"

	"github.com/gin-gonic/gin"
)

type UserHandlerImpl struct {
	usecase usecase.UserUsecase
}

func NewUserHandlerImpl(usecase usecase.UserUsecase) *UserHandlerImpl {
	return &UserHandlerImpl{usecase}
}

func isWeb(ctx *gin.Context) bool {
	return ctx.GetHeader("X-Client-Type") == "web"
}

func (u *UserHandlerImpl) Create(ctx *gin.Context) {
	var request dto.RegisterRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		exception.ErrorHandler(ctx, err)
		return
	}

	result, err := u.usecase.Create(ctx, &request)
	if err != nil {
		exception.ErrorHandler(ctx, err)
		return
	}

	helper.WriteToResponseBody(ctx, http.StatusCreated, dto.WebResponse{
		Status:  "OK",
		Message: "User created successfully",
		Data:    result,
	})
}

func (u *UserHandlerImpl) Login(ctx *gin.Context) {
	var request dto.LoginRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		exception.ErrorHandler(ctx, err)
		return
	}

	result, err := u.usecase.Login(ctx, &request)
	if err != nil {
		exception.ErrorHandler(ctx, err)
		return
	}

	if isWeb(ctx) {
		// Set token ke httpOnly cookie, tidak dikirim di body
		helper.SetAuthCookies(ctx, result.Tokens.AccessToken, result.Tokens.RefreshToken)
		helper.WriteToResponseBody(ctx, http.StatusOK, dto.WebResponse{
			Status:  "OK",
			Message: "Login successful",
			Data: dto.LoginResponse{
				User:   result.User,
				Tokens: nil, // token di cookie, tidak di body
			},
		})
		return
	}

	// Mobile — token di body
	helper.WriteToResponseBody(ctx, http.StatusOK, dto.WebResponse{
		Status:  "OK",
		Message: "Login successful",
		Data:    result,
	})
}

// POST /auth/refresh
func (u *UserHandlerImpl) RefreshToken(ctx *gin.Context) {
	var refreshToken string

	if isWeb(ctx) {
		// Ambil dari cookie
		cookie, err := ctx.Cookie("refresh_token")
		if err != nil || cookie == "" {
			exception.ErrorHandler(ctx, exception.NewUnauthorizedError("refresh token not found"))
			return
		}
		refreshToken = cookie
	} else {
		// Ambil dari body
		var request dto.RefreshRequest
		if err := ctx.ShouldBindJSON(&request); err != nil {
			exception.ErrorHandler(ctx, err)
			return
		}
		if request.RefreshToken == "" {
			exception.ErrorHandler(ctx, exception.NewUnauthorizedError("refresh token is required"))
			return
		}
		refreshToken = request.RefreshToken
	}

	result, err := u.usecase.RefreshToken(ctx, refreshToken)
	if err != nil {
		exception.ErrorHandler(ctx, err)
		return
	}

	if isWeb(ctx) {
		helper.SetAccessTokenCookie(ctx, result.AccessToken)
		helper.WriteToResponseBody(ctx, http.StatusOK, dto.WebResponse{
			Status:  "OK",
			Message: "Token refreshed successfully",
		})
		return
	}

	helper.WriteToResponseBody(ctx, http.StatusOK, dto.WebResponse{
		Status:  "OK",
		Message: "Token refreshed successfully",
		Data:    result,
	})
}

// POST /auth/logout
func (u *UserHandlerImpl) Logout(ctx *gin.Context) {
	if isWeb(ctx) {
		helper.ClearAuthCookies(ctx)
	}
	// Stateless — tidak ada blacklist, cukup clear cookie untuk web
	// Mobile cukup hapus token di sisi client
	helper.WriteToResponseBody(ctx, http.StatusOK, dto.WebResponse{
		Status:  "OK",
		Message: "Logout successful",
	})
}

// GET /auth/me
func (u *UserHandlerImpl) Me(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		exception.ErrorHandler(ctx, exception.NewUnauthorizedError("unauthorized"))
		return
	}

	result, err := u.usecase.Me(ctx, userID.(interface{ String() string }).String())
	if err != nil {
		exception.ErrorHandler(ctx, err)
		return
	}

	helper.WriteToResponseBody(ctx, http.StatusOK, dto.WebResponse{
		Status: "OK",
		Data:   result,
	})
}
