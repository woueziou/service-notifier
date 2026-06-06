package config

import (
	"github.com/caarlos0/env/v11"
)

type Config struct {
	// Server
	Port         int    `env:"PORT" envDefault:"8080"`
	Host         string `env:"HOST" envDefault:"0.0.0.0"`
	ReadTimeout  int    `env:"READ_TIMEOUT_SECONDS" envDefault:"10"`
	WriteTimeout int    `env:"WRITE_TIMEOUT_SECONDS" envDefault:"30"`

	// PostgreSQL
	DatabaseURL string `env:"DATABASE_URL" envDefault:"postgres://postgres:P0stgr3s_Pssw0rd_2024@localhost:5432/notifier?sslmode=disable"`

	// Redis
	RedisHost string `env:"REDIS_HOST" envDefault:"localhost"`
	RedisPort int    `env:"REDIS_PORT" envDefault:"6379"`
	RedisPass string `env:"REDIS_PASSWORD" envDefault:"Rd1s_P@ssw0rd_2024"`
	RedisDB   int    `env:"REDIS_DB" envDefault:"0"`

	// SMTP
	SMTPHost     string `env:"SMTP_HOST" envDefault:"localhost"`
	SMTPPort     int    `env:"SMTP_PORT" envDefault:"1025"`
	SMTPUser     string `env:"SMTP_USER" envDefault:""`
	SMTPPassword string `env:"SMTP_PASSWORD" envDefault:""`
	SMTPFrom     string `env:"SMTP_FROM" envDefault:"notifier@localhost"`

	// Admin
	AdminAPIKey string `env:"ADMIN_API_KEY" envDefault:"admin-key-change-me"`

	// Worker
	WorkerCount int    `env:"WORKER_COUNT" envDefault:"5"`
	ContainerID string `env:"CONTAINER_ID" envDefault:""`

	// Migrations
	MigrationsPath string `env:"MIGRATIONS_PATH" envDefault:"migrations"`
	RunMigrations  bool   `env:"RUN_MIGRATIONS" envDefault:"true"`

	// Redis Streams
	StreamName        string `env:"REDIS_STREAM_NAME" envDefault:"email:jobs"`
	StreamConsumerGroup string `env:"REDIS_CONSUMER_GROUP" envDefault:"notifier-workers"`
	DLQStreamName     string `env:"REDIS_DLQ_STREAM" envDefault:"email:dlq"`
	MaxRetries        int    `env:"MAX_RETRIES" envDefault:"3"`

	// HMAC Auth
	HMACMasterKey string `env:"HMAC_MASTER_KEY" envDefault:""`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
