package handler

import (
	"net/http"
	"titip-jejak-api/internal/dto"
	"titip-jejak-api/internal/exception"
	"titip-jejak-api/internal/helper"
	"titip-jejak-api/internal/usecase"

	"github.com/gin-gonic/gin"
)

type StatsHandlerImpl struct {
	usecase usecase.StatsUsecase
}

func NewStatsHandlerImpl(usecase usecase.StatsUsecase) *StatsHandlerImpl {
	return &StatsHandlerImpl{usecase}
}

// GET /api/v1/stats
func (s *StatsHandlerImpl) GetStats(ctx *gin.Context) {
	result, err := s.usecase.GetStats(ctx)
	if err != nil {
		exception.ErrorHandler(ctx, err)
		return
	}

	helper.WriteToResponseBody(ctx, http.StatusOK, dto.WebResponse{
		Status: "OK",
		Data:   result,
	})
}