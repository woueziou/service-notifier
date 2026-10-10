package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-fuego/fuego"
)

func TestSessionCookiePathDefault(t *testing.T) {
	m := NewAuthModule(nil, nil, nil, "", "")
	if got := m.sessionCookie("id", 60).Path; got != "/" {
		t.Fatalf("path = %q, want /", got)
	}
}

func TestLogoutClearsCookieOnConfiguredPath(t *testing.T) {
	m := NewAuthModule(nil, nil, nil, "", "/notifier")
	s := fuego.NewServer()
	m.Register(s)

	rec := httptest.NewRecorder()
	s.Mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/auth/logout", nil))

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("got %d cookies, want 1", len(cookies))
	}
	c := cookies[0]
	if c.Name != "session" || c.Path != "/notifier" || c.MaxAge >= 0 {
		t.Fatalf("unexpected cookie: %+v", c)
	}
	if set := m.sessionCookie("id", 60); set.Path != c.Path || set.SameSite != http.SameSiteStrictMode || !set.HttpOnly {
		t.Fatalf("login cookie differs from logout cookie: %+v", set)
	}
}
