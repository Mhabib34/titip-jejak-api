package repository

import (
	"context"
	"titip-jejak-api/internal/dto"
	"titip-jejak-api/internal/model"

	"github.com/google/uuid"
)

type MatchRepository interface {
	// Create menyimpan satu match baru. Jika pasangan sudah ada (uq_match_pair),
	// DB akan return error yang diabaikan di worker.
	Create(ctx context.Context, match *model.Match) (*model.Match, error)

	// FindByID mengambil satu match beserta preload kedua report + reporter-nya.
	FindByID(ctx context.Context, id uuid.UUID) (*model.Match, error)

	// FindByUserID mengambil semua match yang berkaitan dengan laporan milik userID.
	FindByUserID(ctx context.Context, userID uuid.UUID, query dto.GetMatchesQuery) ([]model.Match, int64, error)

	// FindPendingNotify mengambil match yang belum dinotifikasi (untuk worker).
	FindPendingNotify(ctx context.Context, limit int) ([]model.Match, error)

	// MarkNotified menandai match sebagai sudah dinotifikasi.
	MarkNotified(ctx context.Context, id uuid.UUID) error

	// FindByReportPair mengecek apakah pasangan sudah pernah di-match.
	FindByReportPair(ctx context.Context, foundID, missingID uuid.UUID) (*model.Match, error)

	// FindActiveReportsByType mengambil laporan aktif berdasarkan tipe (untuk matching).
	FindActiveReportsByType(ctx context.Context, reportType model.ReportType) ([]model.Report, error)

	ExistsByReportPair(ctx context.Context, foundReportID, missingReportID uuid.UUID) (bool, error)
}
