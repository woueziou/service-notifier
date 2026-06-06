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
		Admin:    handler.NewAdminHandler(rdb, cfg.DLQStream, cfg.StreamName, jobRepo, consumerRepo),
	}
}

type ConfigAdapter struct {
	AdminKey     string
	StreamName   string
	DLQStream    string
	MaxRetries   int
	SenderDomain string
}

func (c *ConfigAdapter) JobStream() string {
	return c.StreamName
}

func NewRouter(h *Handlers, consumerRepo *repository.ConsumerRepo, auditRepo *repository.AuditRepo, rateLimiter *service.RateLimiter, metrics *MetricsCollector, cfg *ConfigAdapter) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(LoggerMiddleware)
	r.Use(MetricsMiddleware(metrics))
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))

	// Health and metrics (no auth)
	r.Get("/health", h.Health.Check)
	r.Get("/metrics", MetricsHandler().ServeHTTP)

	// API v1 — consumer-authenticated, rate-limited
	r.Route("/v1", func(r chi.Router) {
		r.Use(BodySizeLimitMiddleware(10 << 20)) // 10 MB max body
		r.Use(IPRateLimitMiddleware(rateLimiter, 120)) // 120 req/min per IP
		r.Use(AuthMiddleware(consumerRepo))
		r.Use(RateLimitMiddleware(rateLimiter, 60)) // 60 req/min per consumer
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

		r.Post("/consumers/{id}/suspend", h.Admin.SuspendConsumer)
		r.Post("/consumers/{id}/reactivate", h.Admin.ReactivateConsumer)

		r.Get("/dlq", h.Admin.ListDLQ)
		r.Post("/dlq/{id}/replay", h.Admin.ReplayDLQ)
		r.Get("/jobs", h.Admin.ListJobs)
	})

	return r
}
