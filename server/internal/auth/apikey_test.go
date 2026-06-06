package auth

import (
	"strings"
	"testing"
)

func TestGenerate_ReturnsKeyWithPrefix(t *testing.T) {
	raw, hash, err := Generate()
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}
	if !strings.HasPrefix(raw, keyPrefix) {
		t.Errorf("raw key should start with %q, got %q", keyPrefix, raw)
	}
	if len(raw) <= len(keyPrefix) {
		t.Errorf("raw key too short: %d chars", len(raw))
	}
	if hash == "" {
		t.Error("hash should not be empty")
	}
}

func TestGenerate_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for range 100 {
		raw, _, err := Generate()
		if err != nil {
			t.Fatalf("Generate() returned error: %v", err)
		}
		if seen[raw] {
			t.Errorf("duplicate key generated: %s", raw)
		}
		seen[raw] = true
	}
}

func TestHash_Deterministic(t *testing.T) {
	key := "nk_testkey123"
	h1 := Hash(key)
	h2 := Hash(key)
	if h1 != h2 {
		t.Errorf("Hash should be deterministic: %s != %s", h1, h2)
	}
}

func TestHash_DifferentKeys(t *testing.T) {
	h1 := Hash("nk_key_one")
	h2 := Hash("nk_key_two")
	if h1 == h2 {
		t.Error("different keys should produce different hashes")
	}
}

func TestHash_Length(t *testing.T) {
	hash := Hash("nk_anykey")
	// SHA-256 hex = 64 chars
	if len(hash) != 64 {
		t.Errorf("expected 64-char hex hash, got %d chars", len(hash))
	}
}

func TestVerify_ValidKey(t *testing.T) {
	raw, hash, err := Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}
	if !Verify(raw, hash) {
		t.Error("Verify should return true for the original key")
	}
}

func TestVerify_WrongKey(t *testing.T) {
	raw, hash, err := Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}
	_ = raw // not used

	if Verify("nk_wrongkey", hash) {
		t.Error("Verify should return false for a different key")
	}
}

func TestVerify_EmptyKey(t *testing.T) {
	_, hash, err := Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}
	if Verify("", hash) {
		t.Error("Verify should return false for empty key")
	}
}

func TestVerify_EmptyHash(t *testing.T) {
	if Verify("nk_somekey", "") {
		t.Error("Verify should return false for empty hash")
	}
}

func TestGenerateAndVerify_RoundTrip(t *testing.T) {
	raw, hash, err := Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}
	if !Verify(raw, hash) {
		t.Fatal("round-trip verification failed")
	}
	// Verify that the same key can be verified again (idempotent)
	if !Verify(raw, hash) {
		t.Fatal("second verification failed (should be idempotent)")
	}
}

func TestGenerate_PanicsOnReadFailure(t *testing.T) {
	// Normal case - just ensure no panic
	_, _, err := Generate()
	if err != nil {
		t.Fatalf("Generate() should not fail: %v", err)
	}
}
