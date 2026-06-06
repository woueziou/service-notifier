package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/flyasky/notifier/internal/config"
	"github.com/flyasky/notifier/internal/engine"
	"github.com/flyasky/notifier/internal/model"
	"github.com/flyasky/notifier/internal/repository"
	"github.com/flyasky/notifier/internal/server"
	"github.com/flyasky/notifier/internal/service"
	"github.com/flyasky/notifier/internal/worker"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// @title        Notifier API
// @version      1.0
// @description  A standalone email dispatch service — single source of truth for email notifications.
// @contact.name  Notifier Team
// @license.name  Proprietary
// @host         localhost:8080
// @schemes      http https
// @securityDefinitions.apikey  BearerAuth
// @in                           header
// @name                         Authorization
func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	// Container identity
	if cfg.ContainerID == "" {
		hostname, _ := os.Hostname()
		cfg.ContainerID = hostname
	}

	// Connect to PostgreSQL
	db, err := server.ConnectDB(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to postgresql")

	// Run migrations
	if cfg.RunMigrations {
		if err := runMigrations(cfg.DatabaseURL, cfg.MigrationsPath); err != nil {
			slog.Error("failed to run migrations", "error", err)
			os.Exit(1)
		}
		slog.Info("database migrations complete")
	} else {
		// Dev fallback: AutoMigrate (creates tables from models)
		if err := db.AutoMigrate(&model.Consumer{}, &model.Job{}, &model.AuditLog{}); err != nil {
			slog.Error("failed to auto-migrate database", "error", err)
			os.Exit(1)
		}
		slog.Info("database auto-migrated (dev mode)")
	}

	// Connect to Redis
	rdb, err := server.ConnectRedis(cfg.RedisHost, cfg.RedisPort, cfg.RedisPass, cfg.RedisDB)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to redis")

	// Ensure Redis stream and consumer group exist
	bgCtx := context.Background()
	if err := server.EnsureStreamGroup(bgCtx, rdb, cfg.StreamName, cfg.StreamConsumerGroup); err != nil {
		slog.Error("failed to create consumer group", "error", err)
		os.Exit(1)
	}
	slog.Info("redis stream consumer group ready")

	// SMTP Engine
	smtpEngine := engine.NewSMTPEngine(
		cfg.SMTPHost,
		cfg.SMTPPort,
		cfg.SMTPUser,
		cfg.SMTPPassword,
		cfg.SMTPFrom,
	)

	// Build handlers & router
	senderDomain := extractDomain(cfg.SMTPFrom)
	adapter := &server.ConfigAdapter{
		AdminKey:     cfg.AdminAPIKey,
		StreamName:   cfg.StreamName,
		DLQStream:    cfg.DLQStreamName,
		MaxRetries:   cfg.MaxRetries,
		SenderDomain: senderDomain,
	}

	handlers := server.NewHandlers(db, rdb, adapter)
	consumerRepo := repository.NewConsumerRepo(db)
	auditRepo := repository.NewAuditRepo(db)
	rateLimiter := service.NewRateLimiter(rdb)
	metrics := server.NewMetricsCollector()
	router := server.NewRouter(handlers, consumerRepo, auditRepo, rateLimiter, metrics, adapter)

	// --- Graceful Shutdown Setup ---
	//
	// On SIGINT/SIGTERM:
	//   1. Cancel worker context → workers finish current message → exit
	//   2. HTTP server shuts down (no new requests)
	//   3. Wait for all workers to finish
	//   4. Close Redis, DB connections
	//   5. Exit

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Start workers
	jobRepo := repository.NewJobRepo(db)
	for i := range cfg.WorkerCount {
		workerID := fmt.Sprintf("%s-worker-%d", cfg.ContainerID, i)
		w := worker.New(
			workerID,
			rdb,
			smtpEngine,
			jobRepo,
			cfg.StreamName,
			cfg.StreamConsumerGroup,
			cfg.DLQStreamName,
			cfg.MaxRetries,
		)
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			slog.Info("worker starting", "id", id)
			w.Run(ctx)
			slog.Info("worker stopped", "id", id)
		}(workerID)
	}
	slog.Info("workers started", "count", cfg.WorkerCount)

	// Start abuse detector
	abuseCfg := service.DefaultAbuseConfig()
	abuseDetector := service.NewAbuseDetector(jobRepo, consumerRepo, abuseCfg)
	wg.Add(1)
	go func() {
		defer wg.Done()
		abuseDetector.Run(ctx)
	}()
	slog.Info("abuse detector started", "interval", abuseCfg.CheckInterval)

	// Start HTTP server (with graceful shutdown)
	srv := server.New(adapter, db, rdb, router)

	// Listen for interrupt signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		slog.Info("shutdown signal received", "signal", sig)

		// Step 1: Stop accepting new work
		cancel()

		// Step 2: Graceful HTTP shutdown
		if err := srv.Shutdown(context.Background()); err != nil {
			slog.Error("http shutdown error", "error", err)
		}
	}()

	// Block until HTTP server stops (either by error or by shutdown signal)
	if err := srv.Start(cfg.Port); err != nil {
		slog.Error("server error", "error", err)
	}

	// Wait for all workers to finish their current message
	slog.Info("waiting for workers to finish...")
	wg.Wait()
	slog.Info("all workers stopped, goodbye")
}

func runMigrations(databaseURL, migrationsPath string) error {
	m, err := migrate.New(
		"file://"+migrationsPath,
		databaseURL,
	)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}

	return nil
}

func extractDomain(from string) string {
	for i := len(from) - 1; i >= 0; i-- {
		if from[i] == '@' {
			return from[i+1:]
		}
	}
	return "localhost"
}
