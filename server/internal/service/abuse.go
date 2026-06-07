package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"woueziou/notifier/internal/repository"
)

// AbuseConfig defines thresholds for automatic consumer suspension.
type AbuseConfig struct {
	MaxBounceRate    float64 // e.g., 0.2 = 20% bounce rate triggers suspension
	MinJobsForBounce int     // Minimum jobs before bounce rate is evaluated (avoid false positives)
	CheckInterval    time.Duration
}

// DefaultAbuseConfig returns sensible defaults for abuse detection.
func DefaultAbuseConfig() AbuseConfig {
	return AbuseConfig{
		MaxBounceRate:    0.2, // 20% bounce rate
		MinJobsForBounce: 10,  // at least 10 jobs before checking
		CheckInterval:    1 * time.Minute,
	}
}

type AbuseDetector struct {
	jobRepo      *repository.JobRepo
	consumerRepo *repository.ConsumerRepo
	cfg          AbuseConfig
}

func NewAbuseDetector(jobRepo *repository.JobRepo, consumerRepo *repository.ConsumerRepo, cfg AbuseConfig) *AbuseDetector {
	return &AbuseDetector{
		jobRepo:      jobRepo,
		consumerRepo: consumerRepo,
		cfg:          cfg,
	}
}

// Run starts the abuse detection loop. Blocks until ctx is cancelled.
func (d *AbuseDetector) Run(ctx context.Context) {
	slog.Info("abuse detector started", "interval", d.cfg.CheckInterval)
	ticker := time.NewTicker(d.cfg.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("abuse detector stopped")
			return
		case <-ticker.C:
			d.checkConsumers(ctx)
		}
	}
}

func (d *AbuseDetector) checkConsumers(ctx context.Context) {
	consumers, err := d.consumerRepo.List(ctx)
	if err != nil {
		slog.Error("abuse detector: failed to list consumers", "error", err)
		return
	}

	for _, consumer := range consumers {
		if consumer.Suspended {
			continue // Already suspended
		}

		// Check bounce rate
		bounceRate, totalJobs, err := d.jobRepo.GetBounceRate(ctx, consumer.ID)
		if err != nil {
			slog.Error("abuse detector: failed to get bounce rate", "consumer_id", consumer.ID, "error", err)
			continue
		}

		if totalJobs < int64(d.cfg.MinJobsForBounce) {
			continue // Not enough data
		}

		if bounceRate > d.cfg.MaxBounceRate {
			slog.Warn("abuse detector: consumer suspended due to high bounce rate",
				"consumer_id", consumer.ID,
				"name", consumer.Name,
				"bounce_rate", bounceRate,
				"total_jobs", totalJobs,
				"threshold", d.cfg.MaxBounceRate,
			)
			if err := d.consumerRepo.Suspend(ctx, consumer.ID, fmt.Sprintf("bounce rate %.1f%% exceeds threshold %.1f%%", bounceRate*100, d.cfg.MaxBounceRate*100)); err != nil {
				slog.Error("abuse detector: failed to suspend consumer", "consumer_id", consumer.ID, "error", err)
			}
		}
	}
}
