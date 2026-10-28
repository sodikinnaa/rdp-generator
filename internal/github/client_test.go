package github

import (
	"crypto/rand"
	"encoding/base64"
	"testing"

	"golang.org/x/crypto/nacl/box"
)

func TestEncryptSecret(t *testing.T) {
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	pubB64 := base64.StdEncoding.EncodeToString(pub[:])
	plainText := "tskey-auth-sample-secret-12345"

	encryptedB64, err := EncryptSecret(plainText, pubB64)
	if err != nil {
		t.Fatalf("EncryptSecret failed: %v", err)
	}

	rawEncrypted, err := base64.StdEncoding.DecodeString(encryptedB64)
	if err != nil {
		t.Fatalf("Decode encrypted failed: %v", err)
	}

	// In anonymous sealed box: box.OpenAnonymous(nil, rawEncrypted, pub, priv)
	decrypted, ok := box.OpenAnonymous(nil, rawEncrypted, pub, priv)
	if !ok {
		t.Fatalf("OpenAnonymous decryption failed")
	}

	if string(decrypted) != plainText {
		t.Fatalf("expected decrypted text %q, got %q", plainText, string(decrypted))
	}
}

func TestEncryptSecretInvalidKey(t *testing.T) {
	_, err := EncryptSecret("plain", "invalid-base64")
	if err == nil {
		t.Fatalf("expected error on invalid base64 key, got nil")
	}

	shortKey := base64.StdEncoding.EncodeToString([]byte("short-key"))
	_, err = EncryptSecret("plain", shortKey)
	if err == nil {
		t.Fatalf("expected error on short key, got nil")
	}
}
