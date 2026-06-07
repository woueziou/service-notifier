package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"

	"woueziou/notifier/internal/engine"
	"woueziou/notifier/internal/repository"
	"github.com/redis/go-redis/v9"
)

type Worker struct {
	id             string
	rdb            *redis.Client
	emailEngine    engine.EmailEngine
	jobRepo        *repository.JobRepo
	streamName     string
	consumerGroup  string
	dlqStream      string
	maxRetries     int
}

func New(id string, rdb *redis.Client, emailEngine engine.EmailEngine, jobRepo *repository.JobRepo,
	streamName, consumerGroup, dlqStream string, maxRetries int) *Worker {
	return &Worker{
		id:            id,
		rdb:           rdb,
		emailEngine:   emailEngine,
		jobRepo:       jobRepo,
		streamName:    streamName,
		consumerGroup: consumerGroup,
		dlqStream:     dlqStream,
		maxRetries:    maxRetries,
	}
}

// Run starts the worker loop. Blocks until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	slog.Info("worker starting", "id", w.id)

	for {
		select {
		case <-ctx.Done():
			slog.Info("worker stopping", "id", w.id)
			return
		default:
		}

		result, err := w.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    w.consumerGroup,
			Consumer: w.id,
			Streams:  []string{w.streamName, ">"},
			Count:    1,
			Block:    5 * time.Second,
		}).Result()

		if err != nil {
			if err == redis.Nil {
				continue // timeout, no messages
			}
			slog.Error("redis xreadgroup error", "worker", w.id, "error", err)
			time.Sleep(1 * time.Second)
			continue
		}

		for _, stream := range result {
			for _, msg := range stream.Messages {
				w.processMessage(ctx, msg)
			}
		}
	}
}

func (w *Worker) processMessage(ctx context.Context, msg redis.XMessage) {
	fields := msg.Values
	jobID := getField(fields, "job_id")
	consumerID := getField(fields, "consumer_id")
	senderEmail := getField(fields, "sender_email")
	toRaw := getField(fields, "to")
	subject := getField(fields, "subject")
	body := getField(fields, "body")
	retryCount := getIntField(fields, "retry_count")
	maxRetries := w.maxRetries
	if mr := getIntField(fields, "max_retries"); mr > 0 {
		maxRetries = mr
	}

	slog.Info("processing message",
		"worker", w.id,
		"job_id", jobID,
		"consumer_id", consumerID,
		"retry", retryCount,
	)

	// Parse recipients
	var to []string
	if err := json.Unmarshal([]byte(toRaw), &to); err != nil {
		slog.Error("failed to parse recipients", "job_id", jobID, "error", err)
		w.ack(ctx, msg.ID)
		return
	}

	// Send email
	err := w.emailEngine.Send(ctx, &engine.EmailMessage{
		From:    senderEmail,
		To:      to,
		Subject: subject,
		Body:    body,
	})

	if err == nil {
		// Success
		if err := w.jobRepo.MarkDelivered(ctx, jobID); err != nil {
			slog.Error("failed to mark delivered", "job_id", jobID, "error", err)
		}
		w.ack(ctx, msg.ID)
		slog.Info("email delivered", "job_id", jobID)
		return
	}

	// Failure
	slog.Error("email send failed", "job_id", jobID, "error", err, "retry", retryCount)

	if retryCount >= maxRetries {
		// Move to DLQ
		w.sendToDLQ(ctx, msg)
		if err := w.jobRepo.MarkFailed(ctx, jobID, err.Error()); err != nil {
			slog.Error("failed to mark job failed", "job_id", jobID, "error", err)
		}
		w.ack(ctx, msg.ID)
		slog.Warn("job moved to DLQ", "job_id", jobID)
		return
	}

	// Re-enqueue with backoff
	w.ack(ctx, msg.ID)
	backoff := time.Duration(math.Pow(2, float64(retryCount+1))) * time.Second

	values := map[string]interface{}{
		"job_id":       jobID,
		"consumer_id":  consumerID,
		"sender_email": senderEmail,
		"to":           toRaw,
		"subject":      subject,
		"body":         body,
		"retry_count":  retryCount + 1,
		"max_retries":  maxRetries,
		"created_at":   time.Now().UTC().Format(time.RFC3339),
		"last_error":   err.Error(),
		"next_attempt": time.Now().UTC().Add(backoff).Format(time.RFC3339),
	}

	if err := w.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: w.streamName,
		Values: values,
	}).Err(); err != nil {
		slog.Error("failed to re-enqueue job", "job_id", jobID, "error", err)
	}
}

func (w *Worker) sendToDLQ(ctx context.Context, msg redis.XMessage) {
	values := make(map[string]interface{})
	for k, v := range msg.Values {
		values[k] = v
	}
	values["failed_at"] = time.Now().UTC().Format(time.RFC3339)

	if err := w.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: w.dlqStream,
		Values: values,
	}).Err(); err != nil {
		slog.Error("failed to add to DLQ", "error", err)
	}
}

func (w *Worker) ack(ctx context.Context, msgID string) {
	if err := w.rdb.XAck(ctx, w.streamName, w.consumerGroup, msgID).Err(); err != nil {
		slog.Error("failed to ack message", "msg_id", msgID, "error", err)
	}
}

func getField(fields map[string]interface{}, key string) string {
	if v, ok := fields[key]; ok && v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func getIntField(fields map[string]interface{}, key string) int {
	if v, ok := fields[key]; ok && v != nil {
		var i int
		if _, err := fmt.Sscanf(fmt.Sprintf("%v", v), "%d", &i); err == nil {
			return i
		}
	}
	return 0
}
