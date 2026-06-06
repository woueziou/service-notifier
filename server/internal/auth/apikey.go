package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
)

const keyPrefix = "nk_"
const keyBytes = 32 // 256-bit key

// Generate creates a new API key and returns the raw key + its SHA-256 hash.
func Generate() (rawKey string, hash string, err error) {
	buf := make([]byte, keyBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generate key: %w", err)
	}
	rawKey = keyPrefix + hex.EncodeToString(buf)
	hash = Hash(rawKey)
	return rawKey, hash, nil
}

// Hash returns the SHA-256 hex digest of a raw key.
func Hash(rawKey string) string {
	sum := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sum[:])
}

// Verify compares a raw key against a stored hash in constant time.
func Verify(rawKey, storedHash string) bool {
	hash := Hash(rawKey)
	return subtle.ConstantTimeCompare([]byte(hash), []byte(storedHash)) == 1
}
