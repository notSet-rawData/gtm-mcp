package auth

import (
	"testing"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := DeriveKey("test-jwt-secret-for-encryption")
	original := `{"access_token":"ya29.abc123","refresh_token":"1//0xyz","expiry":"2025-01-01T00:00:00Z"}`

	encrypted, err := Encrypt(original, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if encrypted == original {
		t.Error("encrypted text should differ from original")
	}

	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != original {
		t.Errorf("roundtrip mismatch: got %q, want %q", decrypted, original)
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	key1 := DeriveKey("correct-key")
	key2 := DeriveKey("wrong-key")

	encrypted, err := Encrypt("sensitive data", key1)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = Decrypt(encrypted, key2)
	if err == nil {
		t.Error("Decrypt with wrong key should fail")
	}
}

func TestEncryptDecryptEmptyString(t *testing.T) {
	key := DeriveKey("test-key")

	encrypted, err := Encrypt("", key)
	if err != nil {
		t.Fatalf("Encrypt empty failed: %v", err)
	}
	if encrypted != "" {
		t.Error("encrypting empty string should return empty string")
	}

	decrypted, err := Decrypt("", key)
	if err != nil {
		t.Fatalf("Decrypt empty failed: %v", err)
	}
	if decrypted != "" {
		t.Error("decrypting empty string should return empty string")
	}
}

func TestDeriveKeyDeterministic(t *testing.T) {
	key1 := DeriveKey("same-secret")
	key2 := DeriveKey("same-secret")

	if len(key1) != 32 {
		t.Errorf("key length = %d, want 32", len(key1))
	}

	for i := range key1 {
		if key1[i] != key2[i] {
			t.Error("DeriveKey should be deterministic")
			break
		}
	}
}

func TestDeriveKeyDifferentSecrets(t *testing.T) {
	key1 := DeriveKey("secret-a")
	key2 := DeriveKey("secret-b")

	same := true
	for i := range key1 {
		if key1[i] != key2[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("different secrets should produce different keys")
	}
}
