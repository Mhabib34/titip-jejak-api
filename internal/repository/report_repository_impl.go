package repository

import (
	"context"
	"fmt"
	"strings"
	"titip-jejak-api/internal/dto"
	"titip-jejak-api/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReportRepositoryImpl struct {
	DB *gorm.DB
}

func NewReportRepository(db *gorm.DB) ReportRepository {
	return &ReportRepositoryImpl{DB: db}
}

func (r *ReportRepositoryImpl) Create(ctx context.Context, report *model.Report) (*model.Report, error) {
	err := r.DB.WithContext(ctx).Create(report).Error
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, report.ID)
}

func (r *ReportRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*model.Report, error) {
	var report model.Report
	err := r.DB.WithContext(ctx).
		Preload("Reporter").
		Where("id = ?", id).
		First(&report).Error
	if err != nil {
		return nil, err
	}
	return &report, nil
}

func (r *ReportRepositoryImpl) FindAll(ctx context.Context, query dto.GetReportsQuery) ([]model.Report, int64, error) {
	var reports []model.Report
	var total int64

	db := r.DB.WithContext(ctx).Model(&model.Report{}).Preload("Reporter")

	// Default status = active kalau tidak disediakan
	if query.Status != nil {
		db = db.Where("status = ?", *query.Status)
	} else {
		db = db.Where("status = ?", model.ReportStatusActive)
	}

	if query.Type != nil {
		db = db.Where("type = ?", *query.Type)
	}
	if query.City != nil && *query.City != "" {
        db = db.Where("LOWER(city) LIKE ?", "%"+strings.ToLower(*query.City)+"%")
	}
	if query.Province != nil && *query.Province != "" {
		db = db.Where("LOWER(province) LIKE ?", "%"+strings.ToLower(*query.Province)+"%")
	}
	if query.Gender != nil {
		db = db.Where("gender = ?", *query.Gender)
	}
	if query.AgeMin != nil {
		db = db.Where("estimated_age >= ?", *query.AgeMin)
	}
	if query.AgeMax != nil {
		db = db.Where("estimated_age <= ?", *query.AgeMax)
	}
	if query.Q != nil && *query.Q != "" {
		search := "%" + strings.ToLower(*query.Q) + "%"
		db = db.Where("LOWER(description) LIKE ? OR LOWER(name) LIKE ?", search, search)
	}

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

	err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&reports).Error
	return reports, total, err
}

func (r *ReportRepositoryImpl) FindByReporterID(ctx context.Context, reporterID uuid.UUID, page, limit int) ([]model.Report, int64, error) {
	var reports []model.Report
	var total int64

	db := r.DB.WithContext(ctx).Model(&model.Report{}).
		Preload("Reporter").
		Where("reporter_id = ?", reporterID)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&reports).Error
	return reports, total, err
}

func (r *ReportRepositoryImpl) FindMapPins(ctx context.Context, query dto.GetMapPinsQuery) ([]model.Report, error) {
	var reports []model.Report

	db := r.DB.WithContext(ctx).
		Select("id, type, gender, estimated_age, city, latitude, longitude, photo_url, created_at").
		Where("status = ? AND latitude IS NOT NULL AND longitude IS NOT NULL", model.ReportStatusActive)

	if query.Type != nil {
		db = db.Where("type = ?", *query.Type)
	}

	// Parse bounds: lat_min,lng_min,lat_max,lng_max
	if query.Bounds != nil && *query.Bounds != "" {
		var latMin, lngMin, latMax, lngMax float64
		_, err := fmt.Sscanf(*query.Bounds, "%f,%f,%f,%f", &latMin, &lngMin, &latMax, &lngMax)
		if err == nil {
			db = db.Where("latitude BETWEEN ? AND ? AND longitude BETWEEN ? AND ?",
				latMin, latMax, lngMin, lngMax)
		}
	}

	err := db.Order("created_at DESC").Find(&reports).Error
	return reports, err
}

func (r *ReportRepositoryImpl) Update(ctx context.Context, report *model.Report) (*model.Report, error) {
	err := r.DB.WithContext(ctx).Save(report).Error
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, report.ID)
}

func (r *ReportRepositoryImpl) UpdatePhotoURL(ctx context.Context, id uuid.UUID, photoURL string) error {
	return r.DB.WithContext(ctx).
		Model(&model.Report{}).
		Where("id = ?", id).
		Update("photo_url", photoURL).Error
}

func (r *ReportRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.DB.WithContext(ctx).Where("id = ?", id).Delete(&model.Report{}).Error
}

func (r *ReportRepositoryImpl) FindActiveByType(ctx context.Context, reportType model.ReportType) ([]model.Report, error) {
	var reports []model.Report
	err := r.DB.WithContext(ctx).
		Where("type = ? AND status = ?", reportType, model.ReportStatusActive).
		Find(&reports).Error
	return reports, err
}
