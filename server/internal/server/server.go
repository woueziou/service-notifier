package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type Server struct {
	cfg    *ConfigAdapter
	router *chi.Mux
	db     *gorm.DB
	rdb    *redis.Client
	srv    *http.Server
}

func New(cfg *ConfigAdapter, db *gorm.DB, rdb *redis.Client, router *chi.Mux) *Server {
	return &Server{
		cfg:    cfg,
		router: router,
		db:     db,
		rdb:    rdb,
	}
}

func ConnectDB(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}

func ConnectRedis(host string, port int, password string, db int) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: password,
		DB:       db,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return rdb, nil
}

// EnsureStreamGroup creates the Redis stream and consumer group if they don't exist.
func EnsureStreamGroup(ctx context.Context, rdb *redis.Client, stream, group string) error {
	err := rdb.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("create consumer group: %w", err)
	}
	return nil
}

// Start begins listening for HTTP requests. Blocks until the server is shut down.
func (s *Server) Start(port int) error {
	addr := fmt.Sprintf(":%d", port)
	s.srv = &http.Server{
		Addr:              addr,
		Handler:           s.router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	slog.Info("server starting", "addr", addr)
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen and serve: %w", err)
	}
	slog.Info("server stopped")
	return nil
}

// Shutdown gracefully stops the HTTP server with a timeout.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.srv == nil {
		return nil
	}
	return s.srv.Shutdown(ctx)
}
