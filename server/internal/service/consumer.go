package service

import (
	"context"
	"fmt"

	"github.com/flyasky/notifier/internal/auth"
	"github.com/flyasky/notifier/internal/model"
	"github.com/flyasky/notifier/internal/repository"
)

type ConsumerService struct {
	repo *repository.ConsumerRepo
}

func NewConsumerService(repo *repository.ConsumerRepo) *ConsumerService {
	return &ConsumerService{repo: repo}
}

func (s *ConsumerService) Create(ctx context.Context, req *model.CreateConsumerRequest, domain string) (*model.CreateConsumerResponse, error) {
	rawKey, hash, err := auth.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	senderEmail := fmt.Sprintf("%s@%s", req.EmailPrefix, domain)

	consumer, err := s.repo.Create(ctx, req.Name, req.EmailPrefix, senderEmail, hash)
	if err != nil {
		return nil, fmt.Errorf("create consumer: %w", err)
	}

	return &model.CreateConsumerResponse{
		ID:          consumer.ID,
		Name:        consumer.Name,
		EmailPrefix: consumer.EmailPrefix,
		SenderEmail: consumer.SenderEmail,
		APIKey:      rawKey, // shown once
	}, nil
}

func (s *ConsumerService) GetByID(ctx context.Context, id string) (*model.Consumer, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ConsumerService) List(ctx context.Context) ([]model.Consumer, error) {
	return s.repo.List(ctx)
}
