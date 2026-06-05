package dto

import (
	"titip-jejak-api/internal/model"
	"time"

	"github.com/google/uuid"
)

// ── Request DTOs ──────────────────────────────────────────────────────────────

type CreateReportRequest struct {
	Type             model.ReportType   `json:"type"               binding:"required,oneof=found missing"`
	Name             *string            `json:"name"`
	Gender           model.ReportGender `json:"gender"             binding:"required,oneof=male female unknown"`
	EstimatedAge     *int               `json:"estimated_age"      binding:"omitempty,min=0,max=120"`
	Description      string             `json:"description"        binding:"required,min=10"`
	LastSeenLocation string             `json:"last_seen_location" binding:"required"`
	City             string             `json:"city"               binding:"required"`
	Province         string             `json:"province"           binding:"required"`
	Latitude         *float64           `json:"latitude"`
	Longitude        *float64           `json:"longitude"`
}

type UpdateReportRequest struct {
	Name             *string              `json:"name"`
	Gender           *model.ReportGender  `json:"gender"             binding:"omitempty,oneof=male female unknown"`
	EstimatedAge     *int                 `json:"estimated_age"      binding:"omitempty,min=0,max=120"`
	Description      *string              `json:"description"        binding:"omitempty,min=10"`
	LastSeenLocation *string              `json:"last_seen_location"`
	City             *string              `json:"city"`
	Province         *string              `json:"province"`
	Latitude         *float64             `json:"latitude"`
	Longitude        *float64             `json:"longitude"`
	Status           *model.ReportStatus  `json:"status"             binding:"omitempty,oneof=active resolved"`
}

type GetReportsQuery struct {
	Type     *model.ReportType   `form:"type"     binding:"omitempty,oneof=found missing"`
	Status   *model.ReportStatus `form:"status"   binding:"omitempty,oneof=active resolved"`
	City     *string             `form:"city"`
	Province *string             `form:"province"`
	Gender   *model.ReportGender `form:"gender"   binding:"omitempty,oneof=male female unknown"`
	AgeMin   *int                `form:"age_min"  binding:"omitempty,min=0"`
	AgeMax   *int                `form:"age_max"  binding:"omitempty,max=120"`
	Q        *string             `form:"q"`
	Page     int                 `form:"page"     binding:"omitempty,min=1"`
	Limit    int                 `form:"limit"    binding:"omitempty,min=1,max=100"`
}

type GetMapPinsQuery struct {
	Type   *model.ReportType `form:"type"   binding:"omitempty,oneof=found missing"`
	Bounds *string           `form:"bounds"`
}

// ── Response DTOs ─────────────────────────────────────────────────────────────

type ReportResponse struct {
	ID               uuid.UUID          `json:"id"`
	ReporterID       uuid.UUID          `json:"reporter_id"`
	Reporter         *UserResponse      `json:"reporter"`
	Type             model.ReportType   `json:"type"`
	Name             *string            `json:"name"`
	Gender           model.ReportGender `json:"gender"`
	EstimatedAge     *int               `json:"estimated_age"`
	PhotoURL         *string            `json:"photo_url"`
	Description      string             `json:"description"`
	LastSeenLocation string             `json:"last_seen_location"`
	City             string             `json:"city"`
	Province         string             `json:"province"`
	Latitude         *float64           `json:"latitude"`
	Longitude        *float64           `json:"longitude"`
	Status           model.ReportStatus `json:"status"`
	WhatsappShareURL string             `json:"whatsapp_share_url"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

type MapPinResponse struct {
	ID           uuid.UUID          `json:"id"`
	Type         model.ReportType   `json:"type"`
	Gender       model.ReportGender `json:"gender"`
	EstimatedAge *int               `json:"estimated_age"`
	City         string             `json:"city"`
	Latitude     float64            `json:"latitude"`
	Longitude    float64            `json:"longitude"`
	PhotoURL     *string            `json:"photo_url"`
	CreatedAt    time.Time          `json:"created_at"`
}

type ReportListData struct {
	Reports []ReportResponse `json:"reports"`
	Meta    Pagination       `json:"meta"`
}

type MapPinsData struct {
	Pins  []MapPinResponse `json:"pins"`
	Total int              `json:"total"`
}

type PhotoUploadResponse struct {
	PhotoURL string `json:"photo_url"`
}
