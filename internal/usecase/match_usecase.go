package usecase

import (
	"context"
	"titip-jejak-api/internal/dto"

	"github.com/google/uuid"
)

type MatchUsecase interface {
	GetAll(ctx context.Context, userID uuid.UUID, query dto.GetMatchesQuery) (*dto.MatchListData, error)
	GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*dto.MatchResponse, error)
}
