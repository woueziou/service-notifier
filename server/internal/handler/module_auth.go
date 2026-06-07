package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"woueziou/notifier/internal/auth"
	"woueziou/notifier/internal/repository"
	"github.com/go-fuego/fuego"
	"github.com/redis/go-redis/v9"
)

// --- Session keys ---

type ctxAdminKey string

const AdminUserContextKey ctxAdminKey = "admin_user"

const sessionPrefix = "admin:session:"
const defaultSessionTTL = 8 * time.Hour

// --- Session types ---

type SessionData struct {
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

// --- Auth Module ---

type AuthModule struct {
	adminRepo *repository.AdminUserRepo
	rdb       *redis.Client
}

func NewAuthModule(adminRepo *repository.AdminUserRepo, rdb *redis.Client) *AuthModule {
	return &AuthModule{
		adminRepo: adminRepo,
		rdb:       rdb,
	}
}

func (m *AuthModule) Register(s *fuego.Server) {
	fuego.Post(s, "/auth/login", m.login)
	fuego.Post(s, "/auth/logout", m.logout)
	fuego.Get(s, "/auth/me", m.me, fuego.OptionMiddleware(SessionAuthMiddleware(m.rdb)))
}

// --- Request/Response types ---

type loginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type loginResponse struct {
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

type meResponse struct {
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

// --- Handlers ---

func (m *AuthModule) login(c fuego.ContextWithBody[loginRequest]) (any, error) {
	req, err := c.Body()
	if err != nil {
		return nil, fuego.BadRequestError{Title: "invalid request body", Detail: err.Error()}
	}

	user, err := m.adminRepo.FindByUsername(c.Context(), req.Username)
	if err != nil {
		return nil, fuego.InternalServerError{Title: "database error"}
	}
	if user == nil {
		return nil, fuego.UnauthorizedError{Title: "invalid username or password"}
	}

	if !auth.VerifyPassword(req.Password, user.PasswordHash) {
		return nil, fuego.UnauthorizedError{Title: "invalid username or password"}
	}

	// Generate session ID
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fuego.InternalServerError{Title: "failed to create session"}
	}

	session := SessionData{
		Username:  user.Username,
		CreatedAt: time.Now(),
	}

	sessionJSON, err := json.Marshal(session)
	if err != nil {
		return nil, fuego.InternalServerError{Title: "failed to marshal session"}
	}

	// Store in Redis with TTL
	key := sessionPrefix + sessionID
	if err := m.rdb.Set(c.Context(), key, sessionJSON, defaultSessionTTL).Err(); err != nil {
		return nil, fuego.InternalServerError{Title: "failed to store session"}
	}

	// Set HTTP-only cookie
	http.SetCookie(c.Response(), &http.Cookie{
		Name:     "session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(defaultSessionTTL.Seconds()),
	})

	return loginResponse{
		Username:  user.Username,
		CreatedAt: user.CreatedAt,
	}, nil
}

func (m *AuthModule) logout(c fuego.ContextNoBody) (any, error) {
	cookie, err := c.Request().Cookie("session")
	if err != nil {
		// No session cookie — nothing to do
		return map[string]string{"status": "ok"}, nil
	}

	// Delete from Redis
	key := sessionPrefix + cookie.Value
	m.rdb.Del(c.Context(), key)

	// Clear cookie
	http.SetCookie(c.Response(), &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	return map[string]string{"status": "logged_out"}, nil
}

func (m *AuthModule) me(c fuego.ContextNoBody) (any, error) {
	session, ok := c.Context().Value(AdminUserContextKey).(*SessionData)
	if !ok || session == nil {
		return nil, fuego.UnauthorizedError{Title: "not authenticated"}
	}

	return meResponse{
		Username:  session.Username,
		CreatedAt: session.CreatedAt,
	}, nil
}

// --- Session middleware ---

// SessionAuthMiddleware validates the session cookie against Redis.
func SessionAuthMiddleware(rdb *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session")
			if err != nil {
				http.Error(w, `{"error":"unauthorized","message":"missing session cookie"}`, http.StatusUnauthorized)
				return
			}

			key := sessionPrefix + cookie.Value
			data, err := rdb.Get(r.Context(), key).Bytes()
			if err != nil {
				http.Error(w, `{"error":"unauthorized","message":"invalid or expired session"}`, http.StatusUnauthorized)
				return
			}

			var session SessionData
			if err := json.Unmarshal(data, &session); err != nil {
				http.Error(w, `{"error":"unauthorized","message":"corrupted session"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), AdminUserContextKey, &session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// --- Helpers ---

func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
