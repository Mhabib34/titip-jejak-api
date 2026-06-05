package usecase

import (
	"context"
	"titip-jejak-api/internal/dto"
)

type StatsUsecase interface {
	GetStats(ctx context.Context) (*dto.StatsData, error)
}