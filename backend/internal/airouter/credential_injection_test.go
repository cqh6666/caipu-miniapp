package airouter

import (
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/credentialcipher"
)

func TestServiceUsesInjectedCredentialBoxWithPreviousKeyring(t *testing.T) {
	t.Parallel()

	oldBox, err := credentialcipher.New(credentialcipher.Key{Version: "old", Secret: "old-secret"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	legacyCiphertext, err := oldBox.Encrypt("provider-key")
	if err != nil {
		t.Fatal(err)
	}
	sharedBox, err := credentialcipher.New(
		credentialcipher.Key{Version: "current", Secret: "current-secret"},
		[]credentialcipher.Key{{Version: "old", Secret: "old-secret"}},
	)
	if err != nil {
		t.Fatal(err)
	}

	service := NewService(nil, sharedBox, nil, nil, nil)
	if service.cipherBox != sharedBox {
		t.Fatal("service did not retain the injected credential box")
	}
	if plain, err := service.cipherBox.Decrypt(legacyCiphertext); err != nil || plain != "provider-key" {
		t.Fatalf("Decrypt() plain=%q error=%v", plain, err)
	}
}
