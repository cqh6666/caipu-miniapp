package credentialcipher

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"strings"
	"testing"
)

func TestVersionedCipherRotationAndRollback(t *testing.T) {
	oldBox, err := New(Key{Version: "2026-01", Secret: "old-secret"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	oldCiphertext, err := oldBox.Encrypt("provider-key")
	if err != nil {
		t.Fatal(err)
	}

	newBox, err := New(Key{Version: "2026-07", Secret: "new-secret"}, []Key{{Version: "2026-01", Secret: "old-secret"}})
	if err != nil {
		t.Fatal(err)
	}
	rotated, changed, err := newBox.Reencrypt(oldCiphertext)
	if err != nil || !changed || !strings.HasPrefix(rotated, "enc:v1:2026-07:") {
		t.Fatalf("rotation changed=%t error=%v ciphertext=%q", changed, err, rotated)
	}
	plain, err := newBox.Decrypt(rotated)
	if err != nil || plain != "provider-key" {
		t.Fatalf("decrypt rotated plain=%q error=%v", plain, err)
	}

	rollbackBox, err := New(Key{Version: "2026-01", Secret: "old-secret"}, []Key{{Version: "2026-07", Secret: "new-secret"}})
	if err != nil {
		t.Fatal(err)
	}
	rolledBack, changed, err := rollbackBox.Reencrypt(rotated)
	if err != nil || !changed || !strings.HasPrefix(rolledBack, "enc:v1:2026-01:") {
		t.Fatalf("rollback changed=%t error=%v ciphertext=%q", changed, err, rolledBack)
	}
}

func TestDecryptSupportsLegacyCiphertext(t *testing.T) {
	legacy := legacyEncrypt(t, "old-secret", "legacy-value")
	box, err := New(Key{Version: "new", Secret: "new-secret"}, []Key{{Version: "old", Secret: "old-secret"}})
	if err != nil {
		t.Fatal(err)
	}
	plain, err := box.Decrypt(legacy)
	if err != nil || plain != "legacy-value" {
		t.Fatalf("legacy decrypt plain=%q error=%v", plain, err)
	}
	if !box.NeedsReencrypt(legacy) {
		t.Fatal("legacy ciphertext should require re-encryption")
	}
}

func legacyEncrypt(t *testing.T, secret, plain string) string {
	t.Helper()
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(plain), nil))
}
