package repository

import (
	"context"
	"fmt"

	"github.com/flyasky/notifier/internal/auth"
	"github.com/flyasky/notifier/internal/model"
	"gorm.io/gorm"
)

type ConsumerRepo struct {
	db *gorm.DB
}

func NewConsumerRepo(db *gorm.DB) *ConsumerRepo {
	return &ConsumerRepo{db: db}
}

func (r *ConsumerRepo) Create(ctx context.Context, name, emailPrefix, senderEmail, apiKeyHash string) (*model.Consumer, error) {
	consumer := &model.Consumer{
		Name:        name,
		EmailPrefix: emailPrefix,
		SenderEmail: senderEmail,
		APIKeyHash:  apiKeyHash,
		Active:      true,
	}
	if err := r.db.WithContext(ctx).Create(consumer).Error; err != nil {
		return nil, fmt.Errorf("create consumer: %w", err)
	}
	return consumer, nil
}

func (r *ConsumerRepo) GetByID(ctx context.Context, id string) (*model.Consumer, error) {
	var consumer model.Consumer
	if err := r.db.WithContext(ctx).First(&consumer, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("get consumer: %w", err)
	}
	return &consumer, nil
}

func (r *ConsumerRepo) List(ctx context.Context) ([]model.Consumer, error) {
	var consumers []model.Consumer
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&consumers).Error; err != nil {
		return nil, fmt.Errorf("list consumers: %w", err)
	}
	return consumers, nil
}

// Authenticate looks up a consumer by raw API key (hashes it first).
func (r *ConsumerRepo) Authenticate(ctx context.Context, rawKey string) (*model.Consumer, error) {
	hash := auth.Hash(rawKey)
	var consumer model.Consumer
	if err := r.db.WithContext(ctx).Where("api_key_hash = ? AND active = ?", hash, true).First(&consumer).Error; err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}
	return &consumer, nil
}
