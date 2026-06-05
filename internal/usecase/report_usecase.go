package usecase

import (
	"context"
	"mime/multipart"
	"titip-jejak-api/internal/dto"

	"github.com/google/uuid"
)

type ReportUsecase interface {
	Create(ctx context.Context, reporterID uuid.UUID, request *dto.CreateReportRequest) (*dto.ReportResponse, error)
	GetAll(ctx context.Context, query dto.GetReportsQuery) (*dto.ReportListData, error)
	GetByID(ctx context.Context, id uuid.UUID) (*dto.ReportResponse, error)
	GetMyReports(ctx context.Context, reporterID uuid.UUID, page, limit int) (*dto.ReportListData, error)
	Update(ctx context.Context, id uuid.UUID, reporterID uuid.UUID, request *dto.UpdateReportRequest) (*dto.ReportResponse, error)
	Delete(ctx context.Context, id uuid.UUID, reporterID uuid.UUID) error
	UploadPhoto(ctx context.Context, id uuid.UUID, reporterID uuid.UUID, file *multipart.FileHeader) (*dto.PhotoUploadResponse, error)
	GetMapPins(ctx context.Context, query dto.GetMapPinsQuery) (*dto.MapPinsData, error)
}
