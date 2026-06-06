package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/flyasky/notifier/internal/model"
	"gorm.io/gorm"
)

type JobRepo struct {
	db *gorm.DB
}

func NewJobRepo(db *gorm.DB) *JobRepo {
	return &JobRepo{db: db}
}

func (r *JobRepo) Create(ctx context.Context, job *model.Job) error {
	if err := r.db.WithContext(ctx).Create(job).Error; err != nil {
		return fmt.Errorf("create job: %w", err)
	}
	return nil
}

func (r *JobRepo) GetByID(ctx context.Context, id string) (*model.Job, error) {
	var job model.Job
	if err := r.db.WithContext(ctx).First(&job, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	return &job, nil
}

func (r *JobRepo) ListByConsumer(ctx context.Context, consumerID string, limit, offset int) ([]model.Job, error) {
	var jobs []model.Job
	query := r.db.WithContext(ctx).Where("consumer_id = ?", consumerID).
		Order("created_at DESC").Limit(limit).Offset(offset)
	if err := query.Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	return jobs, nil
}

func (r *JobRepo) ListAll(ctx context.Context, consumerID, status string, limit, offset int) ([]model.Job, int64, error) {
	var jobs []model.Job
	query := r.db.WithContext(ctx).Model(&model.Job{})

	if consumerID != "" {
		query = query.Where("consumer_id = ?", consumerID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count jobs: %w", err)
	}

	// Fetch page
	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&jobs).Error; err != nil {
		return nil, 0, fmt.Errorf("list jobs: %w", err)
	}

	return jobs, total, nil
}

func (r *JobRepo) MarkDelivered(ctx context.Context, id string) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).Model(&model.Job{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       model.JobStatusDelivered,
			"delivered_at": &now,
		}).Error; err != nil {
		return fmt.Errorf("mark delivered: %w", err)
	}
	return nil
}

func (r *JobRepo) MarkFailed(ctx context.Context, id, errMsg string) error {
	if err := r.db.WithContext(ctx).Model(&model.Job{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status": model.JobStatusFailed,
			"error":  errMsg,
		}).Error; err != nil {
		return fmt.Errorf("mark failed: %w", err)
	}
	return nil
}

// GetBounceRate returns the bounce rate (failed / total) and total job count for a consumer.
func (r *JobRepo) GetBounceRate(ctx context.Context, consumerID string) (float64, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.Job{}).
		Where("consumer_id = ?", consumerID).
		Count(&total).Error; err != nil {
		return 0, 0, fmt.Errorf("count jobs: %w", err)
	}

	if total == 0 {
		return 0, 0, nil
	}

	var failed int64
	if err := r.db.WithContext(ctx).Model(&model.Job{}).
		Where("consumer_id = ? AND status IN ?", consumerID, []model.JobStatus{model.JobStatusFailed, model.JobStatusBounced}).
		Count(&failed).Error; err != nil {
		return 0, 0, fmt.Errorf("count failed: %w", err)
	}

	return float64(failed) / float64(total), total, nil
}
