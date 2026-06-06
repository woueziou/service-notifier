package handler

import (
	"net/http"
	"strconv"

	"github.com/flyasky/notifier/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

type AdminHandler struct {
	rdb        *redis.Client
	dlqStream  string
	jobStream  string
	jobRepo    *repository.JobRepo
}

func NewAdminHandler(rdb *redis.Client, dlqStream, jobStream string, jobRepo *repository.JobRepo) *AdminHandler {
	return &AdminHandler{rdb: rdb, dlqStream: dlqStream, jobStream: jobStream, jobRepo: jobRepo}
}

// ListDLQ returns entries from the dead letter queue.
// @Summary      List dead letters
// @Description  Get entries from the dead letter queue (paginated)
// @Tags         admin
// @Produce      json
// @Param        start  query  string  false  "Stream start ID (default: -)"
// @Param        end    query  string  false  "Stream end ID (default: +)"
// @Param        count  query  int     false  "Max entries (default: 50)"
// @Success      200    {array}  map[string]interface{}
// @Router       /admin/dlq [get]
func (h *AdminHandler) ListDLQ(w http.ResponseWriter, r *http.Request) {
	start := r.URL.Query().Get("start")
	if start == "" {
		start = "-"
	}
	end := r.URL.Query().Get("end")
	if end == "" {
		end = "+"
	}
	count := 50
	if c := r.URL.Query().Get("count"); c != "" {
		if parsed, err := strconv.Atoi(c); err == nil && parsed > 0 {
			count = parsed
		}
	}

	entries, err := h.rdb.XRangeN(r.Context(), h.dlqStream, start, end, int64(count)).Result()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]map[string]interface{}, 0, len(entries))
	for _, e := range entries {
		item := map[string]interface{}{
			"id":     e.ID,
			"fields": e.Values,
		}
		result = append(result, item)
	}

	writeJSON(w, http.StatusOK, result)
}

// ListJobs returns jobs with optional filtering and pagination.
// @Summary      List jobs
// @Description  Get all jobs with optional filtering by consumer and status
// @Tags         admin
// @Produce      json
// @Param        consumer_id  query  string  false  "Filter by consumer ID"
// @Param        status       query  string  false  "Filter by status (pending, delivered, failed, bounced)"
// @Param        limit        query  int     false  "Page size (default: 50)"
// @Param        offset       query  int     false  "Offset (default: 0)"
// @Success      200  {object}  map[string]interface{}
// @Router       /admin/jobs [get]
func (h *AdminHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	consumerID := r.URL.Query().Get("consumer_id")
	status := r.URL.Query().Get("status")
	limit := 50
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	jobs, total, err := h.jobRepo.ListAll(r.Context(), consumerID, status, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":   jobs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// ReplayDLQ re-enqueues a DLQ message back to the main stream.
// @Summary      Replay a dead letter
// @Description  Re-enqueue a message from DLQ to the main job stream
// @Tags         admin
// @Param        id  path  string  true  "DLQ message ID"
// @Success      200  {object}  map[string]string
// @Router       /admin/dlq/{id}/replay [post]
func (h *AdminHandler) ReplayDLQ(w http.ResponseWriter, r *http.Request) {
	msgID := chi.URLParam(r, "id")

	entries, err := h.rdb.XRangeN(r.Context(), h.dlqStream, msgID, msgID, 1).Result()
	if err != nil || len(entries) == 0 {
		writeError(w, http.StatusNotFound, "message not found in DLQ")
		return
	}

	original := entries[0].Values

	// Strip DLQ-specific fields, keep the original job fields
	values := make(map[string]interface{})
	for k, v := range original {
		if k == "failed_at" || k == "last_error" {
			continue
		}
		values[k] = v
	}

	if err := h.rdb.XAdd(r.Context(), &redis.XAddArgs{
		Stream: h.jobStream,
		Values: values,
	}).Err(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Remove from DLQ
	h.rdb.XDel(r.Context(), h.dlqStream, msgID)

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "replayed",
		"message": "re-enqueued to " + h.jobStream,
	})
}
