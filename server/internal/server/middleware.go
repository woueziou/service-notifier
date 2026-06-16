package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"woueziou/notifier/internal/auth"
	"woueziou/notifier/internal/handler"
	"woueziou/notifier/internal/model"
	"woueziou/notifier/internal/repository"
	"woueziou/notifier/internal/service"
)

// --- Custom context keys ---
type ctxKey string

const reqIDKey ctxKey = "request_id"

// --- Middleware: Request ID --------------------------------------------------

// RequestIDMiddleware reads or generates a unique request ID and stores it
// in the context and response header.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = generateID()
		}
		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), reqIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID retrieves the request ID from the context.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(reqIDKey).(string); ok {
		return id
	}
	return ""
}

func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// --- Middleware: Real IP -----------------------------------------------------

// RealIPMiddleware parses X-Forwarded-For and X-Real-IP headers and updates
// the request's RemoteAddr.
func RealIPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			r.RemoteAddr = strings.TrimSpace(parts[0])
		} else if xri := r.Header.Get("X-Real-Ip"); xri != "" {
			r.RemoteAddr = xri
		}
		next.ServeHTTP(w, r)
	})
}

// --- Middleware: Panic Recovery ----------------------------------------------

// RecoveryMiddleware recovers from panics, logs the stack trace, and returns 500.
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				err, ok := rec.(error)
				if !ok {
					err = fmt.Errorf("%v", rec)
				}
				slog.Error("panic recovered",
					"error", err,
					"stack", string(debug.Stack()),
					"request_id", GetRequestID(r.Context()),
				)
				http.Error(w, `{"error":"Internal Server Error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// --- Auth Middleware ---------------------------------------------------------

// AuthMiddleware authenticates requests using either:
//  1. HMAC request signing (X-Consumer-ID, X-Timestamp, X-Signature headers), or
//  2. Bearer token (Authorization: Bearer <api-key>)
//
// HMAC is attempted first; if HMAC headers are present, Bearer fallback is skipped.
func AuthMiddleware(repo *repository.ConsumerRepo, secretProvider repository.HMACSecretProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			consumerID := r.Header.Get("X-Consumer-ID")
			if consumerID != "" {
				authenticateHMAC(w, r, repo, secretProvider, next)
				return
			}

			token := r.Header.Get("Authorization")
			if !strings.HasPrefix(token, "Bearer ") {
				http.Error(w, `{"error":"unauthorized","message":"missing authorization or HMAC headers"}`, http.StatusUnauthorized)
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

func authenticateHMAC(w http.ResponseWriter, r *http.Request, repo *repository.ConsumerRepo, secretProvider repository.HMACSecretProvider, next http.Handler) {
	consumerID := r.Header.Get("X-Consumer-ID")
	timestampStr := r.Header.Get("X-Timestamp")
	signature := r.Header.Get("X-Signature")

	if consumerID == "" || timestampStr == "" || signature == "" {
		http.Error(w, `{"error":"unauthorized","message":"missing HMAC headers"}`, http.StatusUnauthorized)
		return
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"unauthorized","message":"invalid timestamp"}`, http.StatusUnauthorized)
		return
	}

	if !auth.CheckTimestamp(timestamp, auth.DefaultMaxClockSkew) {
		http.Error(w, `{"error":"unauthorized","message":"request expired or clock skew too large"}`, http.StatusUnauthorized)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"unauthorized","message":"cannot read request body"}`, http.StatusUnauthorized)
		return
	}
	r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	var bodyJSON interface{}
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &bodyJSON); err != nil {
			bodyJSON = bodyBytes
		}
	} else {
		bodyJSON = ""
	}

	consumer, err := repo.AuthenticateHMAC(r.Context(), consumerID, timestamp, bodyJSON, signature, secretProvider)
	if err != nil {
		http.Error(w, `{"error":"unauthorized","message":"invalid HMAC signature"}`, http.StatusUnauthorized)
		return
	}

	if consumer.Suspended {
		http.Error(w, `{"error":"forbidden","message":"consumer is suspended"}`, http.StatusForbidden)
		return
	}

	ctx := context.WithValue(r.Context(), handler.ConsumerContextKey, consumer)
	next.ServeHTTP(w, r.WithContext(ctx))
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

// --- Audit ------------------------------------------------------------------

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

// --- Logger -----------------------------------------------------------------

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

// --- Helpers ----------------------------------------------------------------

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

// --- Rate Limiting ----------------------------------------------------------

// RateLimitMiddleware enforces per-consumer rate limits.
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

// --- CORS ------------------------------------------------------------------

// CORSMiddleware allows cross-origin requests from configured origins.
// It handles preflight OPTIONS requests and sets the appropriate headers.
//
// When credentials are required (cookies), the origin must be explicit —
// the wildcard "*" is replaced with the request's Origin header for
// credentialed requests per the CORS spec.
func CORSMiddleware(allowedOrigins string) func(http.Handler) http.Handler {
	// Pre-parse allowed origins into a set for fast lookup
	origins := parseOrigins(allowedOrigins)
	allowAll := allowedOrigins == "*"

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Only process if the request has an Origin header (i.e. cross-origin)
			if origin != "" {
				// Determine the allowed value for this request
				var allowedOrigin string
				switch {
				case allowAll:
					// For credentialed requests, echo back the specific origin.
					// For non-credentialed, we could use "*", but echoing is simpler.
					allowedOrigin = origin
				case origins[origin]:
					allowedOrigin = origin
				default:
					// Origin not allowed — pass through without CORS headers.
					// The browser will reject the response due to missing ACAO header.
					// Still call next so the server can log the request.
					next.ServeHTTP(w, r)
					return
				}

				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")

				// Handle preflight
				if r.Method == http.MethodOptions {
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD, DISPATCH")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-Id, X-Consumer-ID, X-Timestamp, X-Signature")
					w.Header().Set("Access-Control-Max-Age", "86400")
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// parseOrigins splits a comma-separated list of origins into a set.
func parseOrigins(s string) map[string]bool {
	result := make(map[string]bool)
	for _, o := range strings.Split(s, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			result[o] = true
		}
	}
	return result
}
