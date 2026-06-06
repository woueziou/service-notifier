package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	rdb *redis.Client
}

func NewRateLimiter(rdb *redis.Client) *RateLimiter {
	return &RateLimiter{rdb: rdb}
}

// Allow checks if a consumer has exceeded their rate limit.
// Uses a sliding window via Redis sorted sets.
// Returns true if the request is allowed, false if rate-limited.
func (rl *RateLimiter) Allow(ctx context.Context, consumerID string, maxPerMinute int) (bool, error) {
	key := fmt.Sprintf("ratelimit:%s:minute", consumerID)
	now := time.Now().UnixMilli()
	window := now - 60_000 // 1 minute ago

	pipe := rl.rdb.Pipeline()

	// Remove entries outside the window
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", window))

	// Count entries in the window
	countCmd := pipe.ZCard(ctx, key)

	// Add current request
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now),
		Member: fmt.Sprintf("%d", now),
	})

	// Set TTL on the key
	pipe.Expire(ctx, key, 70*time.Second)

	if _, err := pipe.Exec(ctx); err != nil {
		return false, fmt.Errorf("rate limit: %w", err)
	}

	count := countCmd.Val()
	return count < int64(maxPerMinute), nil
}
