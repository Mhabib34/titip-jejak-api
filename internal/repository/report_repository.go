package repository

import (
	"context"
	"titip-jejak-api/internal/dto"
	"titip-jejak-api/internal/model"

	"github.com/google/uuid"
)

type ReportRepository interface {
	Create(ctx context.Context, report *model.Report) (*model.Report, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.Report, error)
	FindAll(ctx context.Context, query dto.GetReportsQuery) ([]model.Report, int64, error)
	FindByReporterID(ctx context.Context, reporterID uuid.UUID, page, limit int) ([]model.Report, int64, error)
	FindMapPins(ctx context.Context, query dto.GetMapPinsQuery) ([]model.Report, error)
	Update(ctx context.Context, report *model.Report) (*model.Report, error)
	UpdatePhotoURL(ctx context.Context, id uuid.UUID, photoURL string) error
	Delete(ctx context.Context, id uuid.UUID) error

	FindActiveByType(ctx context.Context, reportType model.ReportType) ([]model.Report, error)
}
