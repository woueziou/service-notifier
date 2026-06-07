package auth

import (
	"testing"
	"time"
)

func TestGenerateHMACSecret_HasPrefix(t *testing.T) {
	secret, err := GenerateHMACSecret()
	if err != nil {
		t.Fatalf("GenerateHMACSecret() error: %v", err)
	}
	if len(secret) <= len(HMACSecretPrefix) {
		t.Errorf("secret too short: %d chars", len(secret))
	}
	if secret[:len(HMACSecretPrefix)] != HMACSecretPrefix {
		t.Errorf("expected prefix %q, got %q", HMACSecretPrefix, secret[:len(HMACSecretPrefix)])
	}
	// Full length: prefix (4) + 64 hex chars (32 bytes) = 68
	expectedLen := len(HMACSecretPrefix) + 64
	if len(secret) != expectedLen {
		t.Errorf("expected %d chars, got %d", expectedLen, len(secret))
	}
}

func TestSignAndVerify_RoundTrip(t *testing.T) {
	secret := "nsk_test_secret_1234567890abcdef1234567890abcdef12"
	consumerID := "550e8400-e29b-41d4-a716-446655440000"
	body := map[string]interface{}{
		"to":      []interface{}{"user@example.com"},
		"subject": "Hello",
		"body":    "World",
	}

	sig := SignBody(secret, consumerID, 1000000, body)

	if !VerifySignature(secret, consumerID, 1000000, body, sig) {
		t.Error("VerifySignature should return true for correct signature")
	}

	if VerifySignature(secret, consumerID, 1000000, body, "invalid") {
		t.Error("VerifySignature should return false for invalid signature")
	}

	if VerifySignature("wrong_secret", consumerID, 1000000, body, sig) {
		t.Error("VerifySignature should return false for wrong secret")
	}

	if VerifySignature(secret, "wrong-consumer", 1000000, body, sig) {
		t.Error("VerifySignature should return false for wrong consumer ID")
	}

	if VerifySignature(secret, consumerID, 9999999, body, sig) {
		t.Error("VerifySignature should return false for different timestamp")
	}
}

func TestSignAndVerify_WithBytesBody(t *testing.T) {
	secret := "nsk_test_secret"
	consumerID := "consumer-1"
	body := []byte(`{"to":["a@b.com"],"subject":"Hi","body":"there"}`)

	sig := SignBody(secret, consumerID, 1000000, body)

	if !VerifySignature(secret, consumerID, 1000000, body, sig) {
		t.Error("VerifySignature should work with raw JSON bytes body")
	}
}

func TestCheckTimestamp_WithinSkew(t *testing.T) {
	now := time.Now().Unix()
	if !CheckTimestamp(now, 300) {
		t.Error("current timestamp should be valid")
	}
	if !CheckTimestamp(now-120, 300) {
		t.Error("2 minutes ago should be within 5 min skew")
	}
	if !CheckTimestamp(now+120, 300) {
		t.Error("2 minutes in future should be within 5 min skew")
	}
}

func TestCheckTimestamp_OutsideSkew(t *testing.T) {
	now := time.Now().Unix()
	if CheckTimestamp(now-600, 300) {
		t.Error("10 minutes ago should exceed 5 min skew")
	}
	if CheckTimestamp(now+600, 300) {
		t.Error("10 minutes in future should exceed 5 min skew")
	}
}

func TestCanonicalJSON_Deterministic(t *testing.T) {
	body1 := map[string]interface{}{
		"to":      []interface{}{"user@example.com"},
		"subject": "Hello",
		"body":    "World",
	}
	body2 := map[string]interface{}{
		"body":    "World",
		"to":      []interface{}{"user@example.com"},
		"subject": "Hello",
	}

	c1 := canonicalJSON(body1)
	c2 := canonicalJSON(body2)

	if c1 != c2 {
		t.Errorf("canonical JSON should be deterministic:\n  got1: %s\n  got2: %s", c1, c2)
	}
}

func TestParseHMACHeaders(t *testing.T) {
	h := ParseHMACHeaders("consumer-1", "1000000", "abc123")
	if h == nil {
		t.Fatal("expected non-nil headers")
	}
	if h.ConsumerID != "consumer-1" {
		t.Errorf("expected consumer-1, got %s", h.ConsumerID)
	}
	if h.Timestamp != "1000000" {
		t.Errorf("expected 1000000, got %s", h.Timestamp)
	}
	if h.Signature != "abc123" {
		t.Errorf("expected abc123, got %s", h.Signature)
	}
}

func TestParseHMACHeaders_Missing(t *testing.T) {
	if ParseHMACHeaders("", "1000000", "abc123") != nil {
		t.Error("expected nil when consumer ID is empty")
	}
	if ParseHMACHeaders("consumer-1", "", "abc123") != nil {
		t.Error("expected nil when timestamp is empty")
	}
	if ParseHMACHeaders("consumer-1", "1000000", "") != nil {
		t.Error("expected nil when signature is empty")
	}
}

func BenchmarkSignBody(b *testing.B) {
	secret := "nsk_test_secret_1234567890abcdef1234567890abcdef12"
	body := map[string]interface{}{
		"to":      []interface{}{"user@example.com"},
		"subject": "Hello",
		"body":    "World",
	}
	b.ResetTimer()
	for range b.N {
		SignBody(secret, "consumer-1", 1000000, body)
	}
}
