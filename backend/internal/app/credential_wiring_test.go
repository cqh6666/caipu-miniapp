package app

import (
	"strings"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
	"github.com/cqh6666/caipu-miniapp/backend/internal/credentialcipher"
)

func TestBuildCredentialBoxConfiguresCurrentAndPreviousKeys(t *testing.T) {
	t.Parallel()

	oldBox, err := credentialcipher.New(credentialcipher.Key{Version: "old", Secret: "old-secret"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	oldCiphertext, err := oldBox.Encrypt("shared-value")
	if err != nil {
		t.Fatal(err)
	}

	box, err := buildCredentialBox(config.Config{
		CredentialsSecret:       "current-secret",
		CredentialsKeyVersion:   "current",
		CredentialsPreviousKeys: "old=old-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if plain, err := box.Decrypt(oldCiphertext); err != nil || plain != "shared-value" {
		t.Fatalf("Decrypt() plain=%q error=%v", plain, err)
	}
	currentCiphertext, err := box.Encrypt("shared-value")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(currentCiphertext, "enc:v1:current:") {
		t.Fatalf("Encrypt() = %q", currentCiphertext)
	}
}

func TestBuildCredentialBoxRejectsInvalidPreviousKeyring(t *testing.T) {
	t.Parallel()

	_, err := buildCredentialBox(config.Config{
		CredentialsSecret:       "current-secret",
		CredentialsPreviousKeys: "invalid-entry",
	})
	if err == nil || !strings.Contains(err.Error(), "parse previous credential keys") {
		t.Fatalf("error = %v", err)
	}
}
