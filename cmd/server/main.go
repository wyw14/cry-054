package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyw14/cry-054/internal/application"
	"github.com/wyw14/cry-054/internal/config"
	"github.com/wyw14/cry-054/internal/domain"
	"github.com/wyw14/cry-054/internal/platform"
	"github.com/wyw14/cry-054/internal/repository"
	"github.com/wyw14/cry-054/internal/transport/httpapi"
	"go.uber.org/zap"
)

type runtimeStore interface {
	application.Repositories
	application.UnitOfWork
}

func main() {
	if err := run(); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}

func run() error {
	configuration, err := config.Load()
	if err != nil {
		return err
	}
	logger, err := platform.NewLogger(configuration.LogLevel)
	if err != nil {
		return err
	}
	defer func() { _ = logger.Sync() }()
	clock := platform.SystemClock{}
	ids := &platform.MonotonicIDGenerator{}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	store, closeStore, err := openStore(ctx, configuration, clock)
	if err != nil {
		return err
	}
	defer closeStore()
	fileStore, err := platform.NewLocalFileStore(configuration.AttachmentRoot, configuration.MaxUploadBytes, ids)
	if err != nil {
		return err
	}
	notifier := platform.NewLocalNotifier(clock)
	services := httpapi.Services{
		Claims:         application.NewClaimService(store, clock, ids),
		Previews:       application.NewPreviewService(store, clock),
		Settlements:    application.NewSettlementService(store, clock, ids),
		Corrections:    application.NewCorrectionService(store, clock, ids),
		Reviews:        application.NewReviewService(store, clock, ids, notifier),
		Rules:          application.NewRuleService(store, clock),
		Reconciliation: application.NewReconciliationService(store),
		Exports:        application.NewExportService(store, fileStore, clock),
	}
	router := httpapi.NewRouter(services, httpapi.RouterConfig{
		Logger:         logger,
		AllowedOrigins: configuration.AllowedOrigins,
		RequestTimeout: configuration.RequestTimeout,
		Ready:          func() bool { return true },
	})
	server := &http.Server{
		Addr:              configuration.Address,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("HTTP server listening", zap.String("address", configuration.Address))
		serverErrors <- server.ListenAndServe()
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case signalValue := <-signals:
		logger.Info("shutdown requested", zap.String("signal", signalValue.String()))
	case serveErr := <-serverErrors:
		if !errors.Is(serveErr, http.ErrServerClosed) {
			return serveErr
		}
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), configuration.ShutdownTimeout)
	defer shutdownCancel()
	return server.Shutdown(shutdownCtx)
}

func openStore(ctx context.Context, configuration config.Config, clock application.Clock) (runtimeStore, func(), error) {
	if configuration.DatabaseURL != "" {
		store, err := repository.OpenPostgres(ctx, configuration.DatabaseURL)
		if err != nil {
			return nil, func() {}, err
		}
		return store, store.Close, nil
	}
	store := repository.NewMemoryStore()
	now := clock.Now()
	project, err := domain.NewGrantProject("project-care-2026", "困难家庭医疗补助", 2026, domain.MustMoney("50000.00"), now)
	if err != nil {
		return nil, func() {}, err
	}
	claimant, err := domain.NewClaimant("claimant-demo", "演示申请人", "sha256:demo-identity", "enhanced", now)
	if err != nil {
		return nil, func() {}, err
	}
	publishedAt := now
	rule := domain.RuleVersion{
		ID:            "rule-care-2026-v1",
		ProjectID:     project.ID,
		Version:       1,
		EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Cap:           domain.MustMoney("30000.00"),
		Segments: []domain.RateSegment{
			{Threshold: domain.MustMoney("10000.00"), Rate: decimal.RequireFromString("0.80")},
			{Threshold: domain.MustMoney("30000.00"), Rate: decimal.RequireFromString("0.50")},
		},
		Conditions: domain.RuleConditions{
			Categories: map[string]bool{"medical": true, "rehabilitation": true},
			PlanCodes:  map[string]bool{"basic": true, "enhanced": true},
			MinAmount:  domain.MustMoney("1.00"),
		},
		PublishedAt: &publishedAt,
	}
	store.Seed([]domain.GrantProject{project}, []domain.Claimant{claimant}, []domain.RuleVersion{rule})
	return store, func() {}, nil
}
