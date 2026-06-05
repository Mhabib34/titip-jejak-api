package usecase

import (
	"context"
	"titip-jejak-api/internal/dto"
	"titip-jejak-api/internal/exception"
	"titip-jejak-api/internal/repository"
)

type StatsUsecaseImpl struct {
	repo repository.StatsRepository
}

func NewStatsUsecase(repo repository.StatsRepository) StatsUsecase {
	return &StatsUsecaseImpl{repo: repo}
}

func (s *StatsUsecaseImpl) GetStats(ctx context.Context) (*dto.StatsData, error) {
	activeReports, err := s.repo.CountActiveReports(ctx)
	exception.PanicIfError(err)

	volunteers, err := s.repo.CountVolunteers(ctx)
	exception.PanicIfError(err)

	resolvedLast24h, err := s.repo.CountResolvedLast24h(ctx)
	exception.PanicIfError(err)

	uniqueCities, err := s.repo.CountUniqueCities(ctx)
	exception.PanicIfError(err)

	countTotalResolved, err := s.repo.CountTotalResolved(ctx)
	exception.PanicIfError(err)

	return &dto.StatsData{
		ActiveReports:   activeReports,
		TotalVolunteers: volunteers,
		ResolvedLast24h: resolvedLast24h,
		UniqueCities:    uniqueCities,
		TotalResolved:   countTotalResolved,
	}, nil
}