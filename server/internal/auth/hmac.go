package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	// HMACSecretBytes is the length of the generated HMAC secret.
	HMACSecretBytes = 32 // 256-bit key

	// HMACSecretPrefix is the prefix for HMAC secrets returned to consumers.
	HMACSecretPrefix = "nsk_"

	// DefaultMaxClockSkew is the maximum allowed timestamp difference in seconds.
	DefaultMaxClockSkew = 300 // 5 minutes
)

// GenerateHMACSecret creates a new HMAC secret for a consumer.
// Returns the raw secret (shown once) and the encrypted-at-rest version.
func GenerateHMACSecret() (rawSecret string, err error) {
	buf := make([]byte, HMACSecretBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate hmac secret: %w", err)
	}
	rawSecret = HMACSecretPrefix + hex.EncodeToString(buf)
	return rawSecret, nil
}

// SignBody creates an HMAC-SHA256 signature for a request body.
// The message format is: consumerID:timestamp:canonicalBodyJSON
func SignBody(secret string, consumerID string, timestamp int64, body interface{}) string {
	canonical := canonicalJSON(body)
	message := fmt.Sprintf("%s:%d:%s", consumerID, timestamp, canonical)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature checks whether the provided signature matches the expected HMAC.
// Returns true if valid, false otherwise. Uses constant-time comparison.
func VerifySignature(secret string, consumerID string, timestamp int64, body interface{}, signature string) bool {
	expected := SignBody(secret, consumerID, timestamp, body)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(signature)) == 1
}

// CheckTimestamp validates that the request timestamp is within the allowed clock skew.
func CheckTimestamp(timestamp int64, maxSkewSeconds int64) bool {
	now := time.Now().Unix()
	diff := now - timestamp
	if diff < 0 {
		diff = -diff // absolute value
	}
	return diff <= maxSkewSeconds
}

// canonicalJSON produces a sorted, deterministic JSON string for signing.
// This ensures the consumer and server produce identical inputs regardless
// of JSON key ordering.
func canonicalJSON(v interface{}) string {
	switch val := v.(type) {
	case []byte:
		// Already raw bytes — try to normalize as JSON
		var raw interface{}
		if err := json.Unmarshal(val, &raw); err == nil {
			return canonicalJSON(raw)
		}
		return string(val)
	case map[string]interface{}:
		// Sort keys for deterministic output
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var pairs []string
		for _, k := range keys {
			pairs = append(pairs, fmt.Sprintf("%q:%s", k, canonicalJSON(val[k])))
		}
		return "{" + strings.Join(pairs, ",") + "}"
	case []interface{}:
		items := make([]string, len(val))
		for i, item := range val {
			items[i] = canonicalJSON(item)
		}
		return "[" + strings.Join(items, ",") + "]"
	case string:
		b, _ := json.Marshal(val)
		return string(b)
	case float64:
		return fmt.Sprintf("%v", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	default:
		b, _ := json.Marshal(val)
		return string(b)
	}
}

// HMACHeaders contains the request headers used for HMAC authentication.
type HMACHeaders struct {
	ConsumerID string
	Timestamp  string  // raw header value
	Signature  string
}

// ParseHMACHeaders extracts HMAC auth headers from a set of header values.
func ParseHMACHeaders(consumerID, timestamp, signature string) *HMACHeaders {
	if consumerID == "" || timestamp == "" || signature == "" {
		return nil
	}
	return &HMACHeaders{
		ConsumerID: consumerID,
		Timestamp:  timestamp,
		Signature:  signature,
	}
}
