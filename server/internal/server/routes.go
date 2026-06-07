package server

import (
	"net/http"

	chimw "github.com/go-chi/chi/v5/middleware"
	"woueziou/notifier/internal/handler"
	"woueziou/notifier/internal/repository"
	"woueziou/notifier/internal/service"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-fuego/fuego"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ConfigAdapter bridges config values between main.go and the server setup.
type ConfigAdapter struct {
	AdminKey       string
	StreamName     string
	DLQStream      string
	MaxRetries     int
	SenderDomain   string
	SecretProvider repository.HMACSecretProvider
}

// NewFuegoServer creates a fully-wired fuego server with all modules, middleware, and routes.
func NewFuegoServer(db *gorm.DB, rdb *redis.Client, cfg *ConfigAdapter) *fuego.Server {
	// --- Repos ---
	consumerRepo := repository.NewConsumerRepo(db)
	jobRepo := repository.NewJobRepo(db)
	auditRepo := repository.NewAuditRepo(db)

	// --- Services ---
	consumerSvc := service.NewConsumerService(consumerRepo, cfg.SecretProvider)
	dispatchSvc := service.NewDispatchService(jobRepo, rdb, cfg.StreamName, cfg.DLQStream, cfg.MaxRetries, cfg.SenderDomain)
	rateLimiter := service.NewRateLimiter(rdb)

	// --- Fuego server ---
	s := fuego.NewServer(fuego.WithAddr(":8080"))

	// Replace default Stoplight Elements with Scalar API docs UI
	s.OpenAPI.Config.UIHandler = func(specURL string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(scalarHTML(specURL)))
		})
	}

	// --- Global middleware (applied to all routes) ---
	fuego.Use(s, chimw.RequestID)
	fuego.Use(s, chimw.RealIP)
	fuego.Use(s, LoggerMiddleware)
	fuego.Use(s, chimw.Recoverer)

	// --- OpenAPI info ---
	s.OpenAPI.Description().Info.Title = "Notifier API"
	s.OpenAPI.Description().Info.Version = "1.0"
	s.OpenAPI.Description().Info.Description = "A standalone email dispatch service — single source of truth for email notifications."
	s.OpenAPI.Description().Info.Contact = &openapi3.Contact{Name: "Notifier Team"}
	s.OpenAPI.Description().Info.License = &openapi3.License{Name: "Proprietary"}
	s.OpenAPI.Description().Servers = append(s.OpenAPI.Description().Servers, &openapi3.Server{
		URL: "http://localhost:8080",
	})
	s.OpenAPI.Description().Components.SecuritySchemes = openapi3.SecuritySchemes{
		"BearerAuth": &openapi3.SecuritySchemeRef{
			Value: &openapi3.SecurityScheme{
				Type:   "http",
				Scheme: "bearer",
			},
		},
	}

	// --- Modules ---

	// Health (no auth)
	healthModule := handler.NewHealthModule(db, rdb)
	healthModule.Register(s)

	// Dispatch v1 (consumer auth + rate-limit + IP rate-limit + audit)
	dispatchAuth := []func(http.Handler) http.Handler{
		BodySizeLimitMiddleware(10 << 20),
		IPRateLimitMiddleware(rateLimiter, 120),
		AuthMiddleware(consumerRepo, cfg.SecretProvider),
		RateLimitMiddleware(rateLimiter, 60),
		AuditMiddleware(auditRepo),
	}
	dispatchModule := handler.NewDispatchModule(dispatchSvc)
	dispatchModule.Register(s, dispatchAuth...)

	// Admin (admin API key auth)
	adminAuth := []func(http.Handler) http.Handler{
		AdminAuthMiddleware(cfg.AdminKey),
	}
	consumerModule := handler.NewConsumerModule(consumerSvc, cfg.SenderDomain)
	consumerModule.Register(s, adminAuth...)

	adminModule := handler.NewAdminModule(rdb, cfg.DLQStream, cfg.StreamName, jobRepo, consumerRepo)
	adminModule.Register(s, adminAuth...)

	statsModule := handler.NewStatsModule(consumerRepo, jobRepo, rateLimiter)
	statsModule.Register(s, adminAuth...)

	return s
}

// scalarHTML returns a minimal HTML page that renders the Scalar API docs UI from CDN.
func scalarHTML(specURL string) string {
	return `<!doctype html>
<html>
  <head>
    <title>Notifier API</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
  </head>
  <body>
    <script
      id="api-reference"
      data-url="` + specURL + `"
    ></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>`
}
