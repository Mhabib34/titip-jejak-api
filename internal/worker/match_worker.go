package worker

import (
	"context"
	"titip-jejak-api/internal/logger"
	"titip-jejak-api/internal/model"
	"titip-jejak-api/internal/repository"
	"titip-jejak-api/internal/service"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type MatchJob struct {
	ReportID uuid.UUID
}

type MatchWorker struct {
	jobs         chan MatchJob
	reportRepo   repository.ReportRepository
	matchRepo    repository.MatchRepository
	notifRepo    repository.NotificationRepository
	userRepo     repository.UserRepository
	emailService *service.EmailService
	workerCount  int
}

func NewMatchWorker(
	reportRepo repository.ReportRepository,
	matchRepo repository.MatchRepository,
	notifRepo repository.NotificationRepository,
	userRepo repository.UserRepository,
	emailService *service.EmailService,
	workerCount int,
) *MatchWorker {
	if workerCount < 1 {
		workerCount = 3
	}
	return &MatchWorker{
		jobs:         make(chan MatchJob, 100),
		reportRepo:   reportRepo,
		matchRepo:    matchRepo,
		notifRepo:    notifRepo,
		userRepo:     userRepo,
		emailService: emailService,
		workerCount:  workerCount,
	}
}

func (w *MatchWorker) Enqueue(reportID uuid.UUID) {
	log := logger.Get()
	select {
	case w.jobs <- MatchJob{ReportID: reportID}:
		log.Info("job enqueued", "worker", "MatchWorker", "report_id", reportID)
	default:
		log.Warn("job queue full, dropping job", "worker", "MatchWorker", "report_id", reportID)
	}
}

func (w *MatchWorker) Start(ctx context.Context) {
	log := logger.Get()
	eg, ctx := errgroup.WithContext(ctx)

	for i := 0; i < w.workerCount; i++ {
		workerID := i
		eg.Go(func() error {
			log.Info("worker started", "worker", "MatchWorker", "worker_id", workerID)
			for {
				select {
				case <-ctx.Done():
					log.Info("worker stopped", "worker", "MatchWorker", "worker_id", workerID)
					return nil
				case job, ok := <-w.jobs:
					if !ok {
						return nil
					}
					w.processJob(ctx, job)
				}
			}
		})
	}

	if err := eg.Wait(); err != nil {
		log.Error("worker error", "worker", "MatchWorker", "error", err)
	}
}

func (w *MatchWorker) processJob(ctx context.Context, job MatchJob) {
	log := logger.Get()
	log.Info("processing job", "worker", "MatchWorker", "report_id", job.ReportID)

	report, err := w.reportRepo.FindByID(ctx, job.ReportID)
	if err != nil {
		log.Error("report not found", "worker", "MatchWorker", "report_id", job.ReportID, "error", err)
		return
	}

	if report.Status != model.ReportStatusActive {
		log.Info("report not active, skipping", "worker", "MatchWorker", "report_id", job.ReportID, "status", report.Status)
		return
	}

	oppositeType := model.ReportTypeFound
	if report.Type == model.ReportTypeFound {
		oppositeType = model.ReportTypeMissing
	}

	candidates, err := w.reportRepo.FindActiveByType(ctx, oppositeType)
	if err != nil {
		log.Error("failed to fetch candidates", "worker", "MatchWorker", "error", err)
		return
	}

	log.Info("comparing report against candidates", "worker", "MatchWorker", "report_id", job.ReportID, "candidate_count", len(candidates))

	for _, candidate := range candidates {
		w.tryMatch(ctx, *report, candidate)
	}
}

func (w *MatchWorker) tryMatch(ctx context.Context, report, candidate model.Report) {
	log := logger.Get()

	var foundReport, missingReport model.Report
	if report.Type == model.ReportTypeFound {
		foundReport = report
		missingReport = candidate
	} else {
		foundReport = candidate
		missingReport = report
	}

	exists, err := w.matchRepo.ExistsByReportPair(ctx, foundReport.ID, missingReport.ID)
	if err != nil {
		log.Error("ExistsByReportPair error", "worker", "MatchWorker", "error", err)
		return
	}
	if exists {
		return
	}

	score := service.ScoreReports(foundReport, missingReport)
	log.Debug("score calculated", "worker", "MatchWorker", "found_id", foundReport.ID, "missing_id", missingReport.ID, "score", score)

	if score < service.MinMatchScore {
		return
	}

	savedMatch, err := w.matchRepo.Create(ctx, &model.Match{
		FoundReportID:   foundReport.ID,
		MissingReportID: missingReport.ID,
		Score:           score,
		Notified:        false,
	})
	if err != nil {
		log.Error("failed to save match", "worker", "MatchWorker", "error", err)
		return
	}

	log.Info("match saved", "worker", "MatchWorker", "match_id", savedMatch.ID, "score", score)

	w.sendNotifications(ctx, savedMatch, foundReport, missingReport)
}

func (w *MatchWorker) sendNotifications(
	ctx context.Context,
	match *model.Match,
	foundReport, missingReport model.Report,
) {
	log := logger.Get()
	matchID := match.ID

	type recipient struct {
		report  model.Report
		message string
		role    string
	}

	recipients := []recipient{
		{
			report:  foundReport,
			message: formatFinderMessage(match.Score, missingReport),
			role:    "finder",
		},
		{
			report:  missingReport,
			message: formatSeekerMessage(match.Score, foundReport),
			role:    "seeker",
		},
	}

	for _, r := range recipients {
		reportID := r.report.ID

		notif := &model.Notification{
			UserID:   r.report.ReporterID,
			Message:  r.message,
			IsRead:   false,
			ReportID: &reportID,
			MatchID:  &matchID,
		}
		if err := w.notifRepo.Create(ctx, notif); err != nil {
			log.Error("failed to create DB notification", "worker", "MatchWorker", "user_id", r.report.ReporterID, "error", err)
		}

		userID := r.report.ReporterID
		role := r.role
		go w.sendEmail(ctx, match, foundReport, missingReport, userID, role)
	}

	if err := w.matchRepo.MarkNotified(ctx, matchID); err != nil {
		log.Error("failed to mark match as notified", "worker", "MatchWorker", "match_id", matchID, "error", err)
	}
}

func (w *MatchWorker) sendEmail(
	ctx context.Context,
	match *model.Match,
	foundReport, missingReport model.Report,
	userID uuid.UUID,
	role string,
) {
	log := logger.Get()

	if w.emailService == nil {
		log.Warn("emailService is nil, skipping email", "worker", "MatchWorker", "user_id", userID)
		return
	}

	user, err := w.userRepo.FindByID(ctx, userID)
	if err != nil {
		log.Error("user not found for email", "worker", "MatchWorker", "user_id", userID, "error", err)
		return
	}

	payload := service.MatchEmailPayload{
		Match:         match,
		FoundReport:   foundReport,
		MissingReport: missingReport,
		RecipientUser: *user,
		Role:          role,
	}

	if err := w.emailService.SendMatchNotification(ctx, payload); err != nil {
		log.Error("failed to send email", "worker", "MatchWorker", "user_id", userID, "role", role, "match_id", match.ID, "error", err)
		return
	}

	log.Info("email sent", "worker", "MatchWorker", "email", user.Email, "role", role, "match_id", match.ID)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func formatFinderMessage(score int, missingReport model.Report) string {
	name := "seseorang"
	if missingReport.Name != nil && *missingReport.Name != "" {
		name = *missingReport.Name
	}
	return "Laporan penemuan Anda mungkin cocok dengan laporan kehilangan " +
		name + " di " + missingReport.City +
		" (skor: " + itoa(score) + "/100). Cek halaman Matches untuk detail."
}

func formatSeekerMessage(score int, foundReport model.Report) string {
	return "Ada kemungkinan kecocokan untuk laporan Anda! Seseorang ditemukan di " +
		foundReport.City +
		" (skor: " + itoa(score) + "/100). Cek halaman Matches untuk detail."
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [10]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte(n%10) + '0'
		n /= 10
	}
	return string(buf[pos:])
}