package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/flyasky/notifier/internal/config"
	"github.com/flyasky/notifier/internal/engine"
	"github.com/flyasky/notifier/internal/model"
	"github.com/flyasky/notifier/internal/repository"
	"github.com/flyasky/notifier/internal/server"
	"github.com/flyasky/notifier/internal/service"
	"github.com/flyasky/notifier/internal/worker"
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

	// Auto-migrate (dev-friendly; use golang-migrate for production)
	if err := db.AutoMigrate(&model.Consumer{}, &model.Job{}, &model.AuditLog{}); err != nil {
		slog.Error("failed to migrate database", "error", err)
		os.Exit(1)
	}
	slog.Info("database migrated")

	// Connect to Redis
	rdb, err := server.ConnectRedis(cfg.RedisHost, cfg.RedisPort, cfg.RedisPass, cfg.RedisDB)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to redis")

	// Ensure Redis stream and consumer group exist
	ctx := context.Background()
	if err := server.EnsureStreamGroup(ctx, rdb, cfg.StreamName, cfg.StreamConsumerGroup); err != nil {
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
	router := server.NewRouter(handlers, consumerRepo, auditRepo, rateLimiter, adapter)

	// Start workers
	jobRepo := repository.NewJobRepo(db)
	for i := range cfg.WorkerCount {
		w := worker.New(
			fmt.Sprintf("%s-worker-%d", cfg.ContainerID, i),
			rdb,
			smtpEngine,
			jobRepo,
			cfg.StreamName,
			cfg.StreamConsumerGroup,
			cfg.DLQStreamName,
			cfg.MaxRetries,
		)
		go w.Run(ctx)
	}
	slog.Info("workers started", "count", cfg.WorkerCount)

	// Start HTTP server
	srv := server.New(adapter, db, rdb, router)
	if err := srv.Start(cfg.Port); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func extractDomain(from string) string {
	for i := len(from) - 1; i >= 0; i-- {
		if from[i] == '@' {
			return from[i+1:]
		}
	}
	return "localhost"
}
