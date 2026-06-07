package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"woueziou/notifier/internal/engine"
	"woueziou/notifier/internal/model"
	"woueziou/notifier/internal/repository"
	"github.com/go-fuego/fuego"
	"github.com/redis/go-redis/v9"
)

// --- Context keys ---

type ctxAdminKey string

const AdminUserContextKey ctxAdminKey = "admin_user"

// --- Redis keys & TTLs ---

const (
	loginCodePrefix  = "admin:logincode:"
	sessionPrefix    = "admin:session:"
	loginCodeTTL     = 15 * time.Minute
	defaultSessionTTL = 8 * time.Hour
)

// --- Types ---

type SessionData struct {
	Email     string          `json:"email"`
	Role      model.AdminRole `json:"role"`
	CreatedAt time.Time       `json:"created_at"`
}

// --- Auth Module ---

type AuthModule struct {
	adminRepo *repository.AdminUserRepo
	rdb       *redis.Client
	smtp      *engine.SMTPEngine
	fromEmail string
}

func NewAuthModule(adminRepo *repository.AdminUserRepo, rdb *redis.Client, smtp *engine.SMTPEngine, fromEmail string) *AuthModule {
	return &AuthModule{
		adminRepo: adminRepo,
		rdb:       rdb,
		smtp:      smtp,
		fromEmail: fromEmail,
	}
}

func (m *AuthModule) Register(s *fuego.Server) {
	fuego.Post(s, "/auth/request-login", m.requestLogin)
	fuego.Post(s, "/auth/verify-login", m.verifyLogin)
	fuego.Post(s, "/auth/logout", m.logout)
	fuego.Get(s, "/auth/me", m.me, fuego.OptionMiddleware(SessionAuthMiddleware(m.rdb)))
	fuego.Get(s, "/auth/admin-users", m.listAdminUsers, fuego.OptionMiddleware(SessionAuthMiddleware(m.rdb)), fuego.OptionMiddleware(RequireRole(model.RoleAdmin)))
	fuego.Post(s, "/auth/admin-users", m.addAdminUser, fuego.OptionMiddleware(SessionAuthMiddleware(m.rdb)), fuego.OptionMiddleware(RequireRole(model.RoleAdmin)))
	fuego.Delete(s, "/auth/admin-users/{id}", m.deleteAdminUser, fuego.OptionMiddleware(SessionAuthMiddleware(m.rdb)), fuego.OptionMiddleware(RequireRole(model.RoleSuperAdmin)))
}

// --- Request/Response types ---

type requestLoginBody struct {
	Email string `json:"email" validate:"required,email"`
}

type verifyLoginBody struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required"`
}

type loginResponse struct {
	Email string `json:"email"`
}

type meResponse struct {
	Email     string          `json:"email"`
	Role      model.AdminRole `json:"role"`
	CreatedAt time.Time       `json:"created_at"`
}

type adminUserResponse struct {
	ID        string          `json:"id"`
	Email     string          `json:"email"`
	Role      model.AdminRole `json:"role"`
	CreatedAt time.Time       `json:"created_at"`
}

type addAdminUserBody struct {
	Email string          `json:"email" validate:"required,email"`
	Role  model.AdminRole `json:"role" validate:"required"`
}

// --- Handlers ---

func (m *AuthModule) requestLogin(c fuego.ContextWithBody[requestLoginBody]) (any, error) {
	req, err := c.Body()
	if err != nil {
		return nil, fuego.BadRequestError{Title: "invalid request body", Detail: err.Error()}
	}

	// Look up the email — always respond 200 to avoid leaking which emails are registered
	user, _ := m.adminRepo.FindByEmail(c.Context(), req.Email)
	if user == nil {
		// Silently succeed to prevent email enumeration
		return map[string]string{"status": "check_your_email"}, nil
	}

	// Generate a 6-digit code
	code, err := generateLoginCode()
	if err != nil {
		return nil, fuego.InternalServerError{Title: "failed to generate code"}
	}

	// Store in Redis with TTL
	key := loginCodePrefix + req.Email
	if err := m.rdb.Set(c.Context(), key, code, loginCodeTTL).Err(); err != nil {
		return nil, fuego.InternalServerError{Title: "failed to store code"}
	}

	// Send email
	msg := &engine.EmailMessage{
		From:    m.fromEmail,
		To:      []string{req.Email},
		Subject: "Your Notifier Admin Login Code",
		Body:    fmt.Sprintf(loginCodeEmailHTML, code, int(loginCodeTTL.Minutes())),
	}
	if err := m.smtp.Send(c.Context(), msg); err != nil {
		// Log but don't expose — code is stored, user just won't receive it
		// In production, this would need alerting
		return nil, fuego.InternalServerError{Title: "failed to send login email"}
	}

	return map[string]string{"status": "check_your_email"}, nil
}

func (m *AuthModule) verifyLogin(c fuego.ContextWithBody[verifyLoginBody]) (any, error) {
	req, err := c.Body()
	if err != nil {
		return nil, fuego.BadRequestError{Title: "invalid request body", Detail: err.Error()}
	}

	// Validate code from Redis
	key := loginCodePrefix + req.Email
	storedCode, err := m.rdb.Get(c.Context(), key).Result()
	if err != nil || storedCode != req.Code {
		return nil, fuego.UnauthorizedError{Title: "invalid or expired code"}
	}

	// Delete the used code immediately (one-time use)
	m.rdb.Del(c.Context(), key)

	// Re-fetch the user to ensure they still exist
	user, err := m.adminRepo.FindByEmail(c.Context(), req.Email)
	if err != nil || user == nil {
		return nil, fuego.UnauthorizedError{Title: "account not found"}
	}

	// Create session
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fuego.InternalServerError{Title: "failed to create session"}
	}

	session := SessionData{
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: time.Now(),
	}
	sessionJSON, _ := json.Marshal(session)

	sessionKey := sessionPrefix + sessionID
	if err := m.rdb.Set(c.Context(), sessionKey, sessionJSON, defaultSessionTTL).Err(); err != nil {
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

	return loginResponse{Email: user.Email}, nil
}

func (m *AuthModule) logout(c fuego.ContextNoBody) (any, error) {
	cookie, err := c.Request().Cookie("session")
	if err == nil {
		m.rdb.Del(c.Context(), sessionPrefix+cookie.Value)
	}

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
		Email:     session.Email,
		Role:      session.Role,
		CreatedAt: session.CreatedAt,
	}, nil
}

func (m *AuthModule) listAdminUsers(c fuego.ContextNoBody) (any, error) {
	users, err := m.adminRepo.List(c.Context())
	if err != nil {
		return nil, fuego.InternalServerError{Title: "failed to list admin users"}
	}
	result := make([]adminUserResponse, 0, len(users))
	for _, u := range users {
		result = append(result, adminUserResponse{
			ID:        u.ID,
			Email:     u.Email,
			Role:      u.Role,
			CreatedAt: u.CreatedAt,
		})
	}
	return result, nil
}

func (m *AuthModule) addAdminUser(c fuego.ContextWithBody[addAdminUserBody]) (any, error) {
	session := c.Context().Value(AdminUserContextKey).(*SessionData)

	req, err := c.Body()
	if err != nil {
		return nil, fuego.BadRequestError{Title: "invalid request body", Detail: err.Error()}
	}

	// Validate role assignment hierarchy:
	//   super_admin can assign any role
	//   admin can only assign viewer or admin (not super_admin)
	if session.Role != model.RoleSuperAdmin && req.Role == model.RoleSuperAdmin {
		return nil, fuego.ForbiddenError{Title: "cannot assign super_admin role"}
	}

	// Check if email already exists
	existing, _ := m.adminRepo.FindByEmail(c.Context(), req.Email)
	if existing != nil {
		return nil, fuego.ConflictError{Title: "email already registered as admin"}
	}

	// Lookup who created this user (by email)
	createdBy := &session.Email

	user, err := m.adminRepo.Create(c.Context(), req.Email, req.Role, createdBy)
	if err != nil {
		return nil, fuego.InternalServerError{Title: "failed to create admin user"}
	}

	return adminUserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}, nil
}

func (m *AuthModule) deleteAdminUser(c fuego.ContextNoBody) (any, error) {
	id := c.Request().PathValue("id")

	session := c.Context().Value(AdminUserContextKey).(*SessionData)

	// Prevent deleting yourself
	target, err := m.adminRepo.FindByEmail(c.Context(), session.Email)
	if err == nil && target != nil && target.ID == id {
		return nil, fuego.BadRequestError{Title: "cannot delete your own account"}
	}

	if err := m.adminRepo.Delete(c.Context(), id); err != nil {
		return nil, fuego.NotFoundError{Title: "admin user not found"}
	}

	return map[string]string{"status": "deleted"}, nil
}

// --- Middleware ---

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

// RequireRole returns middleware that rejects if the admin's role is below the required level.
// Hierarchy: viewer < admin < super_admin
func RequireRole(minRole model.AdminRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, ok := r.Context().Value(AdminUserContextKey).(*SessionData)
			if !ok || session == nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			if !roleAtLeast(session.Role, minRole) {
				http.Error(w, `{"error":"forbidden","message":"insufficient permissions"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// roleAtLeast checks if the given role is at least the minimum required.
// RequireRoleForWrite returns middleware that enforces the minimum role only on non-GET requests.
// GET requests pass through for all authenticated roles (viewer can view).
// POST/PUT/DELETE require the given minRole.
func RequireRoleForWrite(minRole model.AdminRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// GET requests are read-only — allow all authenticated users
			if r.Method == http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			// Write operations require minimum role
			session, ok := r.Context().Value(AdminUserContextKey).(*SessionData)
			if !ok || session == nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			if !roleAtLeast(session.Role, minRole) {
				http.Error(w, `{"error":"forbidden","message":"insufficient permissions"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func roleAtLeast(role, min model.AdminRole) bool {
	return roleRank(role) >= roleRank(min)
}

func roleRank(role model.AdminRole) int {
	switch role {
	case model.RoleSuperAdmin:
		return 3
	case model.RoleAdmin:
		return 2
	case model.RoleViewer:
		return 1
	default:
		return 0
	}
}

// --- Helpers ---

func generateLoginCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", fmt.Errorf("generate login code: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

const loginCodeEmailHTML = `<!DOCTYPE html>
<html>
<body style="font-family: sans-serif; padding: 32px;">
  <h2>Notifier Admin Login</h2>
  <p>Your login code is:</p>
  <p style="font-size: 32px; font-weight: bold; letter-spacing: 8px; text-align: center; padding: 16px; background: #f3f4f6; border-radius: 8px;">%s</p>
  <p>This code expires in %d minutes.</p>
  <p>If you did not request this code, you can safely ignore this email.</p>
</body>
</html>`
