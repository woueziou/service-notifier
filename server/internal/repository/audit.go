package repository

import (
	"context"
	"fmt"

	"github.com/flyasky/notifier/internal/model"
	"gorm.io/gorm"
)

type AuditRepo struct {
	db *gorm.DB
}

func NewAuditRepo(db *gorm.DB) *AuditRepo {
	return &AuditRepo{db: db}
}

func (r *AuditRepo) Log(ctx context.Context, entry *model.AuditLog) error {
	if err := r.db.WithContext(ctx).Create(entry).Error; err != nil {
		return fmt.Errorf("audit log: %w", err)
	}
	return nil
}
