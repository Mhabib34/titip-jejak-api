package middleware

import (
	"fmt"
	"strings"
	"titip-jejak-api/internal/exception"
	"titip-jejak-api/internal/helper"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var tokenString string

		// Cek header Authorization dulu (mobile)
		authHeader := ctx.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		}

		// Fallback ke cookie (web)
		if tokenString == "" {
			cookie, err := ctx.Cookie("access_token")
			if err == nil {
				tokenString = cookie
			}
		}

		if tokenString == "" {
			panic(exception.NewUnauthorizedError("missing token"))
		}

		claims, err := helper.VerifyAccessToken(tokenString)
		if err != nil {
			panic(exception.NewUnauthorizedError("invalid or expired token"))
		}

		// Set claims ke context
		subRaw, ok := (*claims)["sub"]
		if !ok {
			panic(exception.NewUnauthorizedError("invalid token claims"))
		}

		userID, err := uuid.Parse(fmt.Sprintf("%v", subRaw))
		if err != nil {
			panic(exception.NewUnauthorizedError("invalid token claims"))
		}

		email, _ := (*claims)["email"].(string)

		ctx.Set("user_id", userID)
		ctx.Set("user_email", email)

		ctx.Next()
	}
}
