package usecase

import (
	"context"
	"titip-jejak-api/internal/dto"
	"titip-jejak-api/internal/exception"
	"titip-jejak-api/internal/helper"
	"titip-jejak-api/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MatchUsecaseImpl struct {
	repo repository.MatchRepository
}

func NewMatchUsecase(repo repository.MatchRepository) MatchUsecase {
	return &MatchUsecaseImpl{repo: repo}
}

func (u *MatchUsecaseImpl) GetAll(ctx context.Context, userID uuid.UUID, query dto.GetMatchesQuery) (*dto.MatchListData, error) {
	matches, total, err := u.repo.FindByUserID(ctx, userID, query)
	if err != nil {
		return nil, err
	}

	page := query.Page
	if page < 1 {
		page = 1
	}
	limit := query.Limit
	if limit < 1 {
		limit = 10
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return &dto.MatchListData{
		Matches: helper.ToMatchResponseList(matches),
		Meta: dto.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      int(total),
			TotalPages: totalPages,
		},
	}, nil
}

func (u *MatchUsecaseImpl) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*dto.MatchResponse, error) {
	match, err := u.repo.FindByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, exception.NewNotFoundError("match tidak ditemukan")
		}
		return nil, err
	}

	// Pastikan user hanya bisa lihat match yang berkaitan dengan laporannya
	if match.FoundReport.ReporterID != userID && match.MissingReport.ReporterID != userID {
		return nil, exception.NewForbiddenError("anda tidak memiliki akses ke match ini")
	}

	resp := helper.ToMatchResponse(*match)
	return &resp, nil
}
