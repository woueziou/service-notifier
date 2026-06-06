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
