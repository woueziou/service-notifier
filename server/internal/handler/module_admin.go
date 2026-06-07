package handler

import (
	"net/http"
	"strconv"

	"woueziou/notifier/internal/repository"
	"github.com/go-fuego/fuego"
	"github.com/redis/go-redis/v9"
)

// --- Module: Admin (DLQ, suspend, reactivate) ------------------------------

type AdminModule struct {
	rdb          *redis.Client
	dlqStream    string
	jobStream    string
	jobRepo      *repository.JobRepo
	consumerRepo *repository.ConsumerRepo
}

func NewAdminModule(rdb *redis.Client, dlqStream, jobStream string, jobRepo *repository.JobRepo, consumerRepo *repository.ConsumerRepo) *AdminModule {
	return &AdminModule{
		rdb:          rdb,
		dlqStream:    dlqStream,
		jobStream:    jobStream,
		jobRepo:      jobRepo,
		consumerRepo: consumerRepo,
	}
}

func (m *AdminModule) Register(s *fuego.Server, middlewares ...func(http.Handler) http.Handler) {
	opts := make([]fuego.RouteOption, len(middlewares))
	for i, mw := range middlewares {
		opts[i] = fuego.OptionMiddleware(mw)
	}

	fuego.Post(s, "/admin/consumers/{id}/suspend", m.suspendConsumer, opts...)
	fuego.Post(s, "/admin/consumers/{id}/reactivate", m.reactivateConsumer, opts...)

	fuego.Get(s, "/admin/dlq", m.listDLQ, opts...)
	fuego.Post(s, "/admin/dlq/{id}/replay", m.replayDLQ, opts...)

	fuego.Get(s, "/admin/jobs", m.listJobs, opts...)
	fuego.Get(s, "/admin/jobs/{id}", m.getJob, opts...)
}

func (m *AdminModule) suspendConsumer(c fuego.ContextNoBody) (any, error) {
	id := c.Request().PathValue("id")
	if err := m.consumerRepo.Suspend(c.Context(), id, "manual admin action"); err != nil {
		return nil, err
	}
	return map[string]string{"status": "suspended"}, nil
}

func (m *AdminModule) reactivateConsumer(c fuego.ContextNoBody) (any, error) {
	id := c.Request().PathValue("id")
	if err := m.consumerRepo.Reactivate(c.Context(), id); err != nil {
		return nil, err
	}
	return map[string]string{"status": "reactivated"}, nil
}

func (m *AdminModule) listDLQ(c fuego.ContextNoBody) (any, error) {
	r := c.Request()
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

	entries, err := m.rdb.XRangeN(c.Context(), m.dlqStream, start, end, int64(count)).Result()
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0, len(entries))
	for _, e := range entries {
		item := map[string]interface{}{
			"id":     e.ID,
			"fields": e.Values,
		}
		result = append(result, item)
	}
	return result, nil
}

func (m *AdminModule) replayDLQ(c fuego.ContextNoBody) (any, error) {
	msgID := c.Request().PathValue("id")

	entries, err := m.rdb.XRangeN(c.Context(), m.dlqStream, msgID, msgID, 1).Result()
	if err != nil || len(entries) == 0 {
		return nil, fuego.NotFoundError{Title: "DLQ message not found"}
	}

	original := entries[0].Values
	values := make(map[string]interface{})
	for k, v := range original {
		if k == "failed_at" || k == "last_error" {
			continue
		}
		values[k] = v
	}

	if err := m.rdb.XAdd(c.Context(), &redis.XAddArgs{
		Stream: m.jobStream,
		Values: values,
	}).Err(); err != nil {
		return nil, err
	}

	m.rdb.XDel(c.Context(), m.dlqStream, msgID)

	return map[string]string{
		"status":  "replayed",
		"message": "re-enqueued to " + m.jobStream,
	}, nil
}

func (m *AdminModule) listJobs(c fuego.ContextNoBody) (any, error) {
	r := c.Request()
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

	jobs, total, err := m.jobRepo.ListAll(c.Context(), consumerID, status, limit, offset)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"jobs":   jobs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}, nil
}

func (m *AdminModule) getJob(c fuego.ContextNoBody) (any, error) {
	id := c.Request().PathValue("id")
	job, err := m.jobRepo.GetByID(c.Context(), id)
	if err != nil {
		return nil, fuego.NotFoundError{Title: "Job not found"}
	}
	return job, nil
}
