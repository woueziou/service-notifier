package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"woueziou/notifier/internal/auth"
	"woueziou/notifier/internal/config"
	"woueziou/notifier/internal/engine"
	"woueziou/notifier/internal/model"
	"woueziou/notifier/internal/repository"
	"woueziou/notifier/internal/server"
	"woueziou/notifier/internal/service"
	"woueziou/notifier/internal/worker"

	"gorm.io/gorm"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	if cfg.ContainerID == "" {
		hostname, _ := os.Hostname()
		cfg.ContainerID = hostname
	}

	// --- Database ---
	db, err := server.ConnectDB(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to postgresql")

	if cfg.RunMigrations {
		if err := runMigrations(cfg.DatabaseURL, cfg.MigrationsPath); err != nil {
			slog.Error("failed to run migrations", "error", err)
			os.Exit(1)
		}
		slog.Info("database migrations complete")
	} else {
		if err := db.AutoMigrate(&model.Consumer{}, &model.Job{}, &model.AuditLog{}, &model.AdminUser{}); err != nil {
			slog.Error("failed to auto-migrate database", "error", err)
			os.Exit(1)
		}
		slog.Info("database auto-migrated (dev mode)")
	}

	// --- Redis ---
	rdb, err := server.ConnectRedis(cfg.RedisURL, cfg.RedisHost, cfg.RedisPort, cfg.RedisPass, cfg.RedisDB)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to redis")

	bgCtx := context.Background()
	if err := server.EnsureStreamGroup(bgCtx, rdb, cfg.StreamName, cfg.StreamConsumerGroup); err != nil {
		slog.Error("failed to create consumer group", "error", err)
		os.Exit(1)
	}
	slog.Info("redis stream consumer group ready")

	// --- SMTP Engine ---
	smtpEngine := engine.NewSMTPEngine(
		cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom,
	)

	// --- HMAC secret provider ---
	secretProvider, err := initSecretProvider(cfg.HMACMasterKey)
	if err != nil {
		slog.Error("failed to initialize HMAC secret provider", "error", err)
		os.Exit(1)
	}

	// --- Seed first admin user ---
	if err := seedAdminByEmail(db, cfg.AdminSeedEmail); err != nil {
		slog.Error("failed to seed admin user", "error", err)
		os.Exit(1)
	}

	// --- Build fuego server ---
	senderDomain := extractDomain(cfg.SMTPFrom)
	adapter := &server.ConfigAdapter{
		StreamName:     cfg.StreamName,
		DLQStream:      cfg.DLQStreamName,
		MaxRetries:     cfg.MaxRetries,
		SenderDomain:   senderDomain,
		SecretProvider: secretProvider,
		SMTPEngine:     smtpEngine,
		SMTPFrom:       cfg.SMTPFrom,
		CORSOrigin:     cfg.CORSOrigin,
		Port:           cfg.Port,
	}

	fuegoSrv := server.NewFuegoServer(db, rdb, adapter)

	// --- Graceful Shutdown ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Start workers
	jobRepo := repository.NewJobRepo(db)
	for i := range cfg.WorkerCount {
		workerID := fmt.Sprintf("%s-worker-%d", cfg.ContainerID, i)
		w := worker.New(
			workerID, rdb, smtpEngine, jobRepo,
			cfg.StreamName, cfg.StreamConsumerGroup, cfg.DLQStreamName, cfg.MaxRetries,
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
	abuseDetector := service.NewAbuseDetector(jobRepo, repository.NewConsumerRepo(db), abuseCfg)
	wg.Add(1)
	go func() {
		defer wg.Done()
		abuseDetector.Run(ctx)
	}()
	slog.Info("abuse detector started", "interval", abuseCfg.CheckInterval)

	// Start HTTP server in background (fuego's Run blocks)
	go func() {
		if err := fuegoSrv.Run(); err != nil {
			slog.Error("server error", "error", err)
		}
	}()

	// Wait for signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	slog.Info("shutdown signal received")
	cancel()
	slog.Info("waiting for workers to finish...")
	wg.Wait()
	slog.Info("all workers stopped, goodbye")
}

func runMigrations(databaseURL, migrationsPath string) error {
	m, err := migrate.New("file://"+migrationsPath, databaseURL)
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

func seedAdminByEmail(db *gorm.DB, email string) error {
	if email == "" {
		slog.Warn("ADMIN_SEED_EMAIL not set — no admin user seeded")
		return nil
	}

	repo := repository.NewAdminUserRepo(db)
	ctx := context.Background()

	count, err := repo.Count(ctx)
	if err != nil {
		return fmt.Errorf("check admin users: %w", err)
	}
	if count > 0 {
		slog.Info("admin users already exist, skipping seed")
		return nil
	}

	user, err := repo.Create(ctx, email, model.RoleSuperAdmin, nil)
	if err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}
	slog.Info("first super_admin created", "email", user.Email)
	return nil
}

func initSecretProvider(masterKey string) (repository.HMACSecretProvider, error) {
	if masterKey == "" {
		generated, err := auth.GenerateHMACMasterKey()
		if err != nil {
			return nil, fmt.Errorf("generate hmac master key: %w", err)
		}
		slog.Warn("HMAC_MASTER_KEY not set — generated temporary key (dev mode only)", "key", generated)
		masterKey = generated
	} else {
		if err := auth.ValidateHMACMasterKey(masterKey); err != nil {
			return nil, fmt.Errorf("invalid HMAC_MASTER_KEY: %w", err)
		}
	}
	return repository.NewAESSecretProvider(masterKey), nil
}
