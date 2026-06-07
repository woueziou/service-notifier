package server

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// ConnectDB opens a GORM connection to PostgreSQL with sensible pool settings.
func ConnectDB(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}

// ConnectRedis creates and pings a Redis client.
// If redisURL is non-empty, it is parsed as redis://[user:password@]host[:port][/db].
// Otherwise falls back to individual host/port/password/db fields.
func ConnectRedis(redisURL string, host string, port int, password string, db int) (*redis.Client, error) {
	var opts *redis.Options
	var err error

	if redisURL != "" {
		opts, err = redis.ParseURL(redisURL)
		if err != nil {
			return nil, fmt.Errorf("parse redis url: %w", err)
		}
	} else {
		opts = &redis.Options{
			Addr:     fmt.Sprintf("%s:%d", host, port),
			Password: password,
			DB:       db,
		}
	}

	rdb := redis.NewClient(opts)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return rdb, nil
}

// EnsureStreamGroup creates the Redis stream and consumer group if they don't exist.
func EnsureStreamGroup(ctx context.Context, rdb *redis.Client, stream, group string) error {
	err := rdb.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("create consumer group: %w", err)
	}
	return nil
}
