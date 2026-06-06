package handler

import (
	"encoding/json"
	"net/http"

	"github.com/flyasky/notifier/internal/model"
)

// ConsumerContextKey is the context key used to store the authenticated consumer.
// Must match the key used in server/middleware.go.
var ConsumerContextKey = "consumer"

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, model.ErrorResponse{Error: http.StatusText(status), Message: message})
}

func getConsumer(r *http.Request) *model.Consumer {
	consumer, _ := r.Context().Value(ConsumerContextKey).(*model.Consumer)
	return consumer
}
