package auth

import (
	"testing"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key, err := GenerateHMACMasterKey()
	if err != nil {
		t.Fatalf("GenerateHMACMasterKey() error: %v", err)
	}

	secret := "nsk_test_secret_abcdef1234567890"
	encrypted, err := EncryptSecret(secret, key)
	if err != nil {
		t.Fatalf("EncryptSecret() error: %v", err)
	}

	decrypted, err := DecryptSecret(encrypted, key)
	if err != nil {
		t.Fatalf("DecryptSecret() error: %v", err)
	}

	if decrypted != secret {
		t.Errorf("round-trip failed:\n  expected: %s\n  got:      %s", secret, decrypted)
	}
}

func TestEncrypt_DifferentEachTime(t *testing.T) {
	key, err := GenerateHMACMasterKey()
	if err != nil {
		t.Fatalf("GenerateHMACMasterKey() error: %v", err)
	}

	secret := "nsk_test_secret"
	e1, _ := EncryptSecret(secret, key)
	e2, _ := EncryptSecret(secret, key)

	if e1 == e2 {
		t.Error("encryption should produce different output each time (nonce)")
	}

	d1, _ := DecryptSecret(e1, key)
	d2, _ := DecryptSecret(e2, key)

	if d1 != secret || d2 != secret {
		t.Error("both should decrypt to original secret")
	}
}

func TestDecrypt_WrongKeyFails(t *testing.T) {
	key1, _ := GenerateHMACMasterKey()
	key2, _ := GenerateHMACMasterKey()

	secret := "nsk_test_secret"
	encrypted, _ := EncryptSecret(secret, key1)

	_, err := DecryptSecret(encrypted, key2)
	if err == nil {
		t.Error("decrypt with wrong key should fail")
	}
}

func TestValidateHMACMasterKey_Valid(t *testing.T) {
	key, _ := GenerateHMACMasterKey()
	if err := ValidateHMACMasterKey(key); err != nil {
		t.Errorf("valid key should pass: %v", err)
	}
}

func TestValidateHMACMasterKey_Invalid(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"too short", "abc"},
		{"wrong length", "abcd1234abcd1234abcd1234abcd1234"}, // 16 bytes, not 32
		{"not hex", "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"},
		{"empty", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateHMACMasterKey(tt.key); err == nil {
				t.Errorf("expected error for key %q", tt.key)
			}
		})
	}
}
