package server

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/flyasky/notifier/internal/handler"
	"github.com/flyasky/notifier/internal/model"
	"github.com/flyasky/notifier/internal/repository"
	"github.com/flyasky/notifier/internal/service"
)

// AuthMiddleware validates Bearer tokens against stored hashes.
func AuthMiddleware(repo *repository.ConsumerRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Authorization")
			if !strings.HasPrefix(token, "Bearer ") {
				http.Error(w, `{"error":"unauthorized","message":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}
			rawKey := strings.TrimPrefix(token, "Bearer ")
			consumer, err := repo.Authenticate(r.Context(), rawKey)
			if err != nil {
				http.Error(w, `{"error":"unauthorized","message":"invalid api key"}`, http.StatusUnauthorized)
				return
			}
			if consumer.Suspended {
				http.Error(w, `{"error":"forbidden","message":"consumer is suspended"}`, http.StatusForbidden)
				return
			}
			ctx := context.WithValue(r.Context(), handler.ConsumerContextKey, consumer)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdminAuthMiddleware validates the admin API key for admin routes.
func AdminAuthMiddleware(adminKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Authorization")
			if !strings.HasPrefix(token, "Bearer ") {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			key := strings.TrimPrefix(token, "Bearer ")
			if key != adminKey {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// AuditMiddleware logs API requests to the audit log.
func AuditMiddleware(auditRepo *repository.AuditRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)

			consumerID := ""
			if consumer, ok := r.Context().Value(handler.ConsumerContextKey).(*model.Consumer); ok {
				consumerID = consumer.ID
			}

			entry := &model.AuditLog{
				ConsumerID: consumerID,
				IP:         extractIP(r),
				Endpoint:   r.URL.Path,
				Method:     r.Method,
				StatusCode: sw.status,
			}
			if err := auditRepo.Log(r.Context(), entry); err != nil {
				slog.Error("audit log failed", "error", err)
			}
		})
	}
}

// LoggerMiddleware logs incoming requests.
func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

// RateLimitMiddleware enforces per-consumer rate limits.
// Default: 60 requests/minute per consumer.
func RateLimitMiddleware(rl *service.RateLimiter, maxPerMinute int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			consumer, ok := r.Context().Value(handler.ConsumerContextKey).(*model.Consumer)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			allowed, err := rl.Allow(r.Context(), consumer.ID, maxPerMinute)
			if err != nil {
				slog.Error("rate limit check failed", "consumer_id", consumer.ID, "error", err)
				// Allow on error (fail open)
				next.ServeHTTP(w, r)
				return
			}

			if !allowed {
				w.Header().Set("Retry-After", "60")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"Too Many Requests","message":"rate limit exceeded. Try again in 60 seconds."}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IPRateLimitMiddleware enforces per-IP rate limits for DDoS protection.
func IPRateLimitMiddleware(rl *service.RateLimiter, maxPerMinute int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractIP(r)

			allowed, err := rl.Allow(r.Context(), "ip:"+ip, maxPerMinute)
			if err != nil {
				slog.Error("ip rate limit check failed", "ip", ip, "error", err)
				next.ServeHTTP(w, r)
				return
			}

			if !allowed {
				w.Header().Set("Retry-After", "60")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"Too Many Requests","message":"IP rate limit exceeded."}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// BodySizeLimitMiddleware limits the maximum request body size.
func BodySizeLimitMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
