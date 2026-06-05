package helper

import (
	"fmt"
	"net/url"
	"titip-jejak-api/internal/dto"
	"titip-jejak-api/internal/model"
)

func ToUserResponse(user model.User) *dto.UserResponse {
	return &dto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Phone:     user.Phone,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}
}

func buildWhatsAppShareURL(report model.Report) string {
	name := "Tidak diketahui"
	if report.Name != nil && *report.Name != "" {
		name = *report.Name
	}

	reportType := "Ditemukan"
	if report.Type == model.ReportTypeMissing {
		reportType = "Hilang"
	}

	text := fmt.Sprintf(
		"[TemuKan] %s: %s\nLokasi: %s, %s\nDeskripsi: %s\n\nInfo lebih lanjut di aplikasi TemuKan.",
		reportType,
		name,
		report.LastSeenLocation,
		report.City,
		report.Description,
	)

	return "https://wa.me/?text=" + url.QueryEscape(text)
}

// ToReportResponse mengkonversi model.Report ke dto.ReportResponse
func ToReportResponse(report model.Report) dto.ReportResponse {
	return dto.ReportResponse{
		ID:               report.ID,
		ReporterID:       report.ReporterID,
		Reporter:         ToUserResponse(report.Reporter),
		Type:             report.Type,
		Name:             report.Name,
		Gender:           report.Gender,
		EstimatedAge:     report.EstimatedAge,
		PhotoURL:         report.PhotoURL,
		Description:      report.Description,
		LastSeenLocation: report.LastSeenLocation,
		City:             report.City,
		Province:         report.Province,
		Latitude:         report.Latitude,
		Longitude:        report.Longitude,
		Status:           report.Status,
		WhatsappShareURL: buildWhatsAppShareURL(report),
		CreatedAt:        report.CreatedAt,
		UpdatedAt:        report.UpdatedAt,
	}
}

// ToReportResponseList mengkonversi slice model.Report ke slice dto.ReportResponse
func ToReportResponseList(reports []model.Report) []dto.ReportResponse {
	result := make([]dto.ReportResponse, len(reports))
	for i, r := range reports {
		result[i] = ToReportResponse(r)
	}
	return result
}

// ToMapPinResponse mengkonversi model.Report ke dto.MapPinResponse
func ToMapPinResponse(report model.Report) dto.MapPinResponse {
	lat := 0.0
	lng := 0.0
	if report.Latitude != nil {
		lat = *report.Latitude
	}
	if report.Longitude != nil {
		lng = *report.Longitude
	}

	return dto.MapPinResponse{
		ID:           report.ID,
		Type:         report.Type,
		Gender:       report.Gender,
		EstimatedAge: report.EstimatedAge,
		City:         report.City,
		Latitude:     lat,
		Longitude:    lng,
		PhotoURL:     report.PhotoURL,
		CreatedAt:    report.CreatedAt,
	}
}

// ToMapPinResponseList mengkonversi slice model.Report ke slice dto.MapPinResponse
func ToMapPinResponseList(reports []model.Report) []dto.MapPinResponse {
	result := make([]dto.MapPinResponse, len(reports))
	for i, r := range reports {
		result[i] = ToMapPinResponse(r)
	}
	return result
}

// ── Match Mappers ─────────────────────────────────────────────────────────────

func ToMatchResponse(match model.Match) dto.MatchResponse {
	return dto.MatchResponse{
		ID:            match.ID,
		Score:         match.Score,
		FoundReport:   ToReportResponse(match.FoundReport),
		MissingReport: ToReportResponse(match.MissingReport),
		Notified:      match.Notified,
		CreatedAt:     match.CreatedAt,
	}
}

func ToMatchResponseList(matches []model.Match) []dto.MatchResponse {
	result := make([]dto.MatchResponse, 0, len(matches))
	for _, m := range matches {
		result = append(result, ToMatchResponse(m))
	}
	return result
}

// ── Notification Mappers ──────────────────────────────────────────────────────

func ToNotificationResponse(n model.Notification) dto.NotificationResponse {
	return dto.NotificationResponse{
		ID:        n.ID,
		Message:   n.Message,
		IsRead:    n.IsRead,
		ReportID:  n.ReportID,
		MatchID:   n.MatchID,
		CreatedAt: n.CreatedAt,
	}
}

func ToNotificationResponseList(notifications []model.Notification) []dto.NotificationResponse {
	result := make([]dto.NotificationResponse, 0, len(notifications))
	for _, n := range notifications {
		result = append(result, ToNotificationResponse(n))
	}
	return result
}
