package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"status": "ok"}

	writeJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	var decoded map[string]string
	if err := json.NewDecoder(w.Body).Decode(&decoded); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if decoded["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", decoded["status"])
	}
}

func TestWriteJSON_DifferentStatuses(t *testing.T) {
	tests := []struct {
		status int
		data   interface{}
	}{
		{http.StatusOK, map[string]int{"count": 42}},
		{http.StatusCreated, map[string]string{"id": "abc"}},
		{http.StatusAccepted, map[string]string{"status": "queued"}},
		{http.StatusBadRequest, map[string]string{"error": "bad input"}},
		{http.StatusNotFound, map[string]string{"error": "not found"}},
		{http.StatusTooManyRequests, map[string]string{"error": "rate limited"}},
		{http.StatusInternalServerError, map[string]string{"error": "server error"}},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			w := httptest.NewRecorder()
			writeJSON(w, tt.status, tt.data)

			if w.Code != tt.status {
				t.Errorf("expected status %d, got %d", tt.status, w.Code)
			}
		})
	}
}

func TestWriteJSON_NilData(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// nil should serialize to "null"
	if w.Body.String() != "null\n" {
		t.Errorf("expected body 'null\\n', got %q", w.Body.String())
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusBadRequest, "invalid email format")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp["error"] != "Bad Request" {
		t.Errorf("expected error 'Bad Request', got '%s'", resp["error"])
	}
	if resp["message"] != "invalid email format" {
		t.Errorf("expected message 'invalid email format', got '%s'", resp["message"])
	}
}

func TestGetConsumer_NoConsumer(t *testing.T) {
	r := httptest.NewRequest("GET", "/test", nil)
	consumer := getConsumer(r)
	if consumer != nil {
		t.Error("expected nil consumer when not in context")
	}
}
