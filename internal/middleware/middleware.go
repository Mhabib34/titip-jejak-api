package middleware

import (
	"titip-jejak-api/internal/exception"

	"github.com/gin-gonic/gin"
)

func ErrorRecovery() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				exception.ErrorHandler(ctx, r)
				ctx.Abort()
			}
		}()
		ctx.Next()
	}
}
