package repository

import (
	"context"
	"titip-jejak-api/internal/dto"
	"titip-jejak-api/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MatchRepositoryImpl struct {
	DB *gorm.DB
}

func NewMatchRepository(db *gorm.DB) MatchRepository {
	return &MatchRepositoryImpl{DB: db}
}

func (r *MatchRepositoryImpl) Create(ctx context.Context, match *model.Match) (*model.Match, error) {
	err := r.DB.WithContext(ctx).Create(match).Error
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, match.ID)
}

func (r *MatchRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*model.Match, error) {
	var match model.Match
	err := r.DB.WithContext(ctx).
		Preload("FoundReport.Reporter").
		Preload("MissingReport.Reporter").
		Where("id = ?", id).
		First(&match).Error
	if err != nil {
		return nil, err
	}
	return &match, nil
}

func (r *MatchRepositoryImpl) FindByUserID(ctx context.Context, userID uuid.UUID, query dto.GetMatchesQuery) ([]model.Match, int64, error) {
	var matches []model.Match
	var total int64

	minScore := query.MinScore
	if minScore == 0 {
		minScore = 60 // default dari spec
	}

	// Ambil match yang found_report atau missing_report milik user
	db := r.DB.WithContext(ctx).
		Joins("JOIN reports fr ON fr.id = matches.found_report_id").
		Joins("JOIN reports mr ON mr.id = matches.missing_report_id").
		Where("(fr.reporter_id = ? OR mr.reporter_id = ?) AND matches.score >= ?", userID, userID, minScore).
		Model(&model.Match{})

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := query.Page
	if page < 1 {
		page = 1
	}
	limit := query.Limit
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	err := db.
		Preload("FoundReport.Reporter").
		Preload("MissingReport.Reporter").
		Order("matches.score DESC").
		Offset(offset).Limit(limit).
		Find(&matches).Error

	return matches, total, err
}

func (r *MatchRepositoryImpl) FindPendingNotify(ctx context.Context, limit int) ([]model.Match, error) {
	var matches []model.Match
	err := r.DB.WithContext(ctx).
		Preload("FoundReport.Reporter").
		Preload("MissingReport.Reporter").
		Where("notified = false").
		Order("score DESC").
		Limit(limit).
		Find(&matches).Error
	return matches, err
}

func (r *MatchRepositoryImpl) MarkNotified(ctx context.Context, id uuid.UUID) error {
	return r.DB.WithContext(ctx).
		Model(&model.Match{}).
		Where("id = ?", id).
		Update("notified", true).Error
}

func (r *MatchRepositoryImpl) FindByReportPair(ctx context.Context, foundID, missingID uuid.UUID) (*model.Match, error) {
	var match model.Match
	err := r.DB.WithContext(ctx).
		Where("found_report_id = ? AND missing_report_id = ?", foundID, missingID).
		First(&match).Error
	if err != nil {
		return nil, err
	}
	return &match, nil
}

func (r *MatchRepositoryImpl) FindActiveReportsByType(ctx context.Context, reportType model.ReportType) ([]model.Report, error) {
	var reports []model.Report
	err := r.DB.WithContext(ctx).
		Where("status = ? AND type = ?", model.ReportStatusActive, reportType).
		Find(&reports).Error
	return reports, err
}

func (r *MatchRepositoryImpl) ExistsByReportPair(ctx context.Context, foundReportID, missingReportID uuid.UUID) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&model.Match{}).
		Where("found_report_id = ? AND missing_report_id = ?", foundReportID, missingReportID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
