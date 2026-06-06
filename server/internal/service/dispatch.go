package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/flyasky/notifier/internal/model"
	"github.com/flyasky/notifier/internal/repository"
	"github.com/redis/go-redis/v9"
)

type DispatchService struct {
	jobRepo      *repository.JobRepo
	rdb          *redis.Client
	streamName   string
	dlqStream    string
	maxRetries   int
	senderDomain string
}

func NewDispatchService(jobRepo *repository.JobRepo, rdb *redis.Client, streamName, dlqStream string, maxRetries int, senderDomain string) *DispatchService {
	return &DispatchService{
		jobRepo:      jobRepo,
		rdb:          rdb,
		streamName:   streamName,
		dlqStream:    dlqStream,
		maxRetries:   maxRetries,
		senderDomain: senderDomain,
	}
}

// Enqueue validates and adds an email job to the Redis stream.
func (s *DispatchService) Enqueue(ctx context.Context, consumer *model.Consumer, req *model.SendRequest) (*model.SendResponse, error) {
	// Validate
	if len(req.To) == 0 {
		return nil, fmt.Errorf("at least one recipient required")
	}

	// Create job record
	job := &model.Job{
		ConsumerID: consumer.ID,
		Status:     model.JobStatusPending,
		To:         mustMarshalJSON(req.To),
		Subject:    req.Subject,
		Body:       req.Body,
	}
	if err := s.jobRepo.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}

	// Enqueue to Redis Stream
	values := map[string]interface{}{
		"job_id":       job.ID,
		"consumer_id":  consumer.ID,
		"sender_email": consumer.SenderEmail,
		"to":           job.To,
		"subject":      job.Subject,
		"body":         job.Body,
		"retry_count":  0,
		"max_retries":  s.maxRetries,
		"created_at":   time.Now().UTC().Format(time.RFC3339),
	}

	if err := s.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: s.streamName,
		Values: values,
	}).Err(); err != nil {
		return nil, fmt.Errorf("enqueue to stream: %w", err)
	}

	return &model.SendResponse{
		JobID:  job.ID,
		Status: "queued",
	}, nil
}

// GetJob returns a job by ID.
func (s *DispatchService) GetJob(ctx context.Context, id string) (*model.Job, error) {
	return s.jobRepo.GetByID(ctx, id)
}

func mustMarshalJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("marshal json: %v", err))
	}
	return string(b)
}
