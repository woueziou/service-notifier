package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

	// Unique member per request (prevents dedup when multiple requests hit same millisecond)
	member := fmt.Sprintf("%d:%s", now, randomHex(8))

	pipe := rl.rdb.Pipeline()

	// Remove entries outside the window
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", window))

	// Count entries in the window
	countCmd := pipe.ZCard(ctx, key)

	// Add current request
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now),
		Member: member,
	})

	// Set TTL on the key
	pipe.Expire(ctx, key, 70*time.Second)

	if _, err := pipe.Exec(ctx); err != nil {
		return false, fmt.Errorf("rate limit: %w", err)
	}

	count := countCmd.Val()
	return count < int64(maxPerMinute), nil
}

// GetCurrentCount reads the current request count in the sliding window
// without adding a new entry. Used by the admin stats endpoint.
func (rl *RateLimiter) GetCurrentCount(ctx context.Context, consumerID string) (int64, error) {
	key := fmt.Sprintf("ratelimit:%s:minute", consumerID)
	now := time.Now().UnixMilli()
	window := now - 60_000

	// Clean up old entries and count
	pipe := rl.rdb.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", window))
	countCmd := pipe.ZCard(ctx, key)
	pipe.Expire(ctx, key, 70*time.Second)

	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("get rate limit count: %w", err)
	}

	return countCmd.Val(), nil
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
