package repository

import (
	"context"
	"fmt"

	"woueziou/notifier/internal/auth"
	"woueziou/notifier/internal/model"
	"gorm.io/gorm"
)

// HMACSecretProvider defines how HMAC secrets are encrypted/decrypted.
// The implementation receives the raw secret and returns it encrypted for storage,
// and vice versa. In production, this is backed by AES-256-GCM with a master key.
type HMACSecretProvider interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(encrypted string) (string, error)
}

type aesSecretProvider struct {
	masterKey string
}

func NewAESSecretProvider(masterKey string) HMACSecretProvider {
	return &aesSecretProvider{masterKey: masterKey}
}

func (p *aesSecretProvider) Encrypt(plaintext string) (string, error) {
	return auth.EncryptSecret(plaintext, p.masterKey)
}

func (p *aesSecretProvider) Decrypt(encrypted string) (string, error) {
	return auth.DecryptSecret(encrypted, p.masterKey)
}

type ConsumerRepo struct {
	db *gorm.DB
}

func NewConsumerRepo(db *gorm.DB) *ConsumerRepo {
	return &ConsumerRepo{db: db}
}

func (r *ConsumerRepo) Create(ctx context.Context, name, emailPrefix, senderEmail, apiKeyHash, hmacSecretEncrypted string) (*model.Consumer, error) {
	consumer := &model.Consumer{
		Name:                name,
		EmailPrefix:         emailPrefix,
		SenderEmail:         senderEmail,
		APIKeyHash:          apiKeyHash,
		HMACSecretEncrypted: hmacSecretEncrypted,
		Active:              true,
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

// AuthenticateHMAC looks up a consumer by ID and verifies the provided HMAC signature
// against the consumer's stored HMAC secret. Returns the consumer if valid.
func (r *ConsumerRepo) AuthenticateHMAC(ctx context.Context, consumerID string, timestamp int64, body interface{}, signature string, secretProvider HMACSecretProvider) (*model.Consumer, error) {
	consumer, err := r.GetByID(ctx, consumerID)
	if err != nil {
		return nil, fmt.Errorf("authenticate hmac: %w", err)
	}

	if !consumer.Active || consumer.Suspended {
		return nil, fmt.Errorf("authenticate hmac: consumer not active or suspended")
	}

	if consumer.HMACSecretEncrypted == "" {
		return nil, fmt.Errorf("authenticate hmac: consumer has no HMAC secret")
	}

	secret, err := secretProvider.Decrypt(consumer.HMACSecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("authenticate hmac: decrypt secret: %w", err)
	}

	if !auth.VerifySignature(secret, consumerID, timestamp, body, signature) {
		return nil, fmt.Errorf("authenticate hmac: invalid signature")
	}

	return consumer, nil
}

func (r *ConsumerRepo) Suspend(ctx context.Context, id, reason string) error {
	return r.db.WithContext(ctx).Model(&model.Consumer{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"suspended": true,
			"active":    false,
		}).Error
}

func (r *ConsumerRepo) Reactivate(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&model.Consumer{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"suspended": false,
			"active":    true,
		}).Error
}
