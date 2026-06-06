package service

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// TestRateLimiter_Allow_Integration is a manual test that requires a real Redis.
//
// To run:
//   REDIS_TEST=1 go test ./internal/service/ -run TestRateLimiter_Allow_Integration -v
//
// Prerequisites: Redis running on localhost:6379 with password "Rd1s_P@ssw0rd_2024"
func TestRateLimiter_Allow_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "Rd1s_P@ssw0rd_2024",
		DB:       0,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skipf("redis not available: %v", err)
	}
	defer rdb.Close()

	rl := NewRateLimiter(rdb)
	consumerID := "test-consumer-rate"

	// Clean up any previous test data
	rdb.Del(context.Background(), "ratelimit:test-consumer-rate:minute")

	// Allow 5 requests per minute
	maxPerMinute := 5

	// First 5 requests should be allowed
	for i := 0; i < maxPerMinute; i++ {
		allowed, err := rl.Allow(context.Background(), consumerID, maxPerMinute)
		if err != nil {
			t.Fatalf("Allow() returned error on request %d: %v", i, err)
		}
		if !allowed {
			t.Errorf("request %d should be allowed (under limit)", i)
		}
	}

	// 6th request should be rate-limited
	allowed, err := rl.Allow(context.Background(), consumerID, maxPerMinute)
	if err != nil {
		t.Fatalf("Allow() returned error: %v", err)
	}
	if allowed {
		t.Error("6th request should be rate-limited")
	}

	// Clean up
	rdb.Del(context.Background(), "ratelimit:test-consumer-rate:minute")
}

func TestRateLimiter_Allow_Twice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "Rd1s_P@ssw0rd_2024",
		DB:       0,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skipf("redis not available: %v", err)
	}
	defer rdb.Close()

	rl := NewRateLimiter(rdb)
	consumerID := "test-consumer-rate-2"

	rdb.Del(context.Background(), "ratelimit:test-consumer-rate-2:minute")

	// Same consumer, high limit — should always be allowed
	for i := 0; i < 10; i++ {
		allowed, err := rl.Allow(context.Background(), consumerID, 100)
		if err != nil {
			t.Fatalf("Allow() returned error: %v", err)
		}
		if !allowed {
			t.Errorf("request %d should be allowed", i)
		}
	}

	rdb.Del(context.Background(), "ratelimit:test-consumer-rate-2:minute")
}

func TestRateLimiter_Allow_DifferentConsumers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "Rd1s_P@ssw0rd_2024",
		DB:       0,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skipf("redis not available: %v", err)
	}
	defer rdb.Close()

	rl := NewRateLimiter(rdb)

	// Clean up
	rdb.Del(context.Background(), "ratelimit:consumer-a:minute")
	rdb.Del(context.Background(), "ratelimit:consumer-b:minute")

	// Both consumers should be able to make requests independently
	allowedA, _ := rl.Allow(context.Background(), "consumer-a", 1)
	if !allowedA {
		t.Error("consumer-a first request should be allowed")
	}

	allowedB, _ := rl.Allow(context.Background(), "consumer-b", 1)
	if !allowedB {
		t.Error("consumer-b first request should be allowed")
	}

	// consumer-a should now be rate-limited (limit was 1)
	allowedA2, _ := rl.Allow(context.Background(), "consumer-a", 1)
	if allowedA2 {
		t.Error("consumer-a second request should be rate-limited")
	}

	// consumer-b should also be rate-limited
	allowedB2, _ := rl.Allow(context.Background(), "consumer-b", 1)
	if allowedB2 {
		t.Error("consumer-b second request should be rate-limited")
	}

	rdb.Del(context.Background(), "ratelimit:consumer-a:minute")
	rdb.Del(context.Background(), "ratelimit:consumer-b:minute")
}

func TestRateLimiter_New(t *testing.T) {
	// Just verify constructor works
	rl := NewRateLimiter(nil)
	if rl == nil {
		t.Fatal("NewRateLimiter should not return nil")
	}
}
