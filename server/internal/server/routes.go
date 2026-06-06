package server

import (
	"time"

	"github.com/flyasky/notifier/internal/handler"
	"github.com/flyasky/notifier/internal/repository"
	"github.com/flyasky/notifier/internal/service"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Handlers struct {
	Consumer *handler.ConsumerHandler
	Dispatch *handler.DispatchHandler
	Health   *handler.HealthHandler
	Admin    *handler.AdminHandler
}

func NewHandlers(db *gorm.DB, rdb *redis.Client, cfg *ConfigAdapter) *Handlers {
	// Repos
	consumerRepo := repository.NewConsumerRepo(db)
	jobRepo := repository.NewJobRepo(db)

	// Services
	consumerSvc := service.NewConsumerService(consumerRepo)
	dispatchSvc := service.NewDispatchService(jobRepo, rdb, cfg.StreamName, cfg.DLQStream, cfg.MaxRetries, cfg.SenderDomain)

	// Handlers
	return &Handlers{
		Consumer: handler.NewConsumerHandler(consumerSvc, cfg.SenderDomain),
		Dispatch: handler.NewDispatchHandler(dispatchSvc),
		Health:   handler.NewHealthHandler(db, rdb),
		Admin:    handler.NewAdminHandler(rdb, cfg.DLQStream),
	}
}

type ConfigAdapter struct {
	AdminKey     string
	StreamName   string
	DLQStream    string
	MaxRetries   int
	SenderDomain string
}

func NewRouter(h *Handlers, consumerRepo *repository.ConsumerRepo, auditRepo *repository.AuditRepo, cfg *ConfigAdapter) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(LoggerMiddleware)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))

	// Health (no auth)
	r.Get("/health", h.Health.Check)

	// API v1 — consumer-authenticated
	r.Route("/v1", func(r chi.Router) {
		r.Use(AuthMiddleware(consumerRepo))
		r.Use(AuditMiddleware(auditRepo))
		r.Post("/send", h.Dispatch.Send)
		r.Get("/jobs/{id}", h.Dispatch.GetJob)
	})

	// Admin routes — admin API key
	r.Route("/admin", func(r chi.Router) {
		r.Use(AdminAuthMiddleware(cfg.AdminKey))

		r.Get("/consumers", h.Consumer.List)
		r.Get("/consumers/{id}", h.Consumer.GetByID)
		r.Post("/consumers", h.Consumer.Create)

		r.Get("/dlq", h.Admin.ListDLQ)
		r.Post("/dlq/{id}/replay", h.Admin.ReplayDLQ)
		// Consumer-scoped job listing is not implemented yet
	})

	return r
}
