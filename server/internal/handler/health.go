package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewHealthHandler(db *gorm.DB, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, rdb: rdb}
}

// @Summary      Health check
// @Description  Check if the service is running and its dependencies are healthy
// @Tags         system
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /health [get]
func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	dbOK := true
	if sqlDB, err := h.db.DB(); err != nil || sqlDB.Ping() != nil {
		dbOK = false
	}

	redisOK := true
	if err := h.rdb.Ping(context.Background()).Err(); err != nil {
		slog.Error("redis health check failed", "error", err)
		redisOK = false
	}

	status := http.StatusOK
	if !dbOK || !redisOK {
		status = http.StatusServiceUnavailable
	}

	writeJSON(w, status, map[string]interface{}{
		"status":  http.StatusText(status),
		"db":      dbOK,
		"redis":   redisOK,
	})
}
