//go:build wireinject
// +build wireinject

package wire

import (
	"context"
	"os"
	"strconv"

	"titip-jejak-api/config"
	"titip-jejak-api/internal/handler"
	"titip-jejak-api/internal/logger"
	"titip-jejak-api/internal/repository"
	"titip-jejak-api/internal/router"
	"titip-jejak-api/internal/service"
	"titip-jejak-api/internal/usecase"
	"titip-jejak-api/internal/worker"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/wire"
)

// ── Provider Sets ─────────────────────────────────────────────────────────────

var infraSet = wire.NewSet(
	config.Load,
	config.NewDB,
	config.NewCloudinary,
	provideValidator,
	provideEmailService,
)

var repositorySet = wire.NewSet(
	repository.NewUserRepository,
	repository.NewReportRepository,
	repository.NewMatchRepository,
	repository.NewNotificationRepository,
	repository.NewStatsRepository,
)

var workerSet = wire.NewSet(
	provideMatchWorker,
)

var usecaseSet = wire.NewSet(
	usecase.NewUserUsecase,
	provideReportUsecase,
	usecase.NewMatchUsecase,
	usecase.NewNotificationUsecase,
	usecase.NewStatsUsecase,
)

var handlerSet = wire.NewSet(
	handler.NewUserHandlerImpl,
	wire.Bind(new(handler.UserHandler), new(*handler.UserHandlerImpl)),

	handler.NewReportHandlerImpl,
	wire.Bind(new(handler.ReportHandler), new(*handler.ReportHandlerImpl)),

	handler.NewMatchHandlerImpl,
	wire.Bind(new(handler.MatchHandler), new(*handler.MatchHandlerImpl)),

	handler.NewNotificationHandlerImpl,
	wire.Bind(new(handler.NotificationHandler), new(*handler.NotificationHandlerImpl)),

	handler.NewStatsHandlerImpl,
	wire.Bind(new(handler.StatsHandler), new(*handler.StatsHandlerImpl)),
)

// ── Individual Providers ──────────────────────────────────────────────────────

func provideValidator() *validator.Validate {
	return validator.New()
}

func provideEmailService() *service.EmailService {
	return service.NewEmailService(
		os.Getenv("RESEND_API_KEY"),
		os.Getenv("RESEND_FROM_EMAIL"),
		os.Getenv("APP_URL"),
	)
}

func provideMatchWorker(
	reportRepo repository.ReportRepository,
	matchRepo repository.MatchRepository,
	notifRepo repository.NotificationRepository,
	userRepo repository.UserRepository,
	emailSvc *service.EmailService,
) (*worker.MatchWorker, func(), error) {
	log := logger.Get()

	count := 3 // default
    if v := os.Getenv("WORKER_COUNT"); v != "" {
        if n, err := strconv.Atoi(v); err == nil && n > 0 {
            count = n
        }
    }

    mw := worker.NewMatchWorker(reportRepo, matchRepo, notifRepo, userRepo, emailSvc, count)

	ctx, cancel := context.WithCancel(context.Background())
	go mw.Start(ctx)
	log.Info("match worker started")

	cleanup := func() {
		cancel()
		log.Info("match worker stopped")
	}

	return mw, cleanup, nil
}

func provideReportUsecase(
	repo repository.ReportRepository,
	validate *validator.Validate,
	cld *cloudinary.Cloudinary,
	mw *worker.MatchWorker,
) usecase.ReportUsecase {
	return usecase.NewReportUsecase(repo, validate, cld, mw)
}

// ── Injector ──────────────────────────────────────────────────────────────────

func InitializeApp() (*gin.Engine, func(), error) {
	wire.Build(
		infraSet,
		repositorySet,
		workerSet,
		usecaseSet,
		handlerSet,
		router.SetupRouter,
	)
	return nil, nil, nil
}