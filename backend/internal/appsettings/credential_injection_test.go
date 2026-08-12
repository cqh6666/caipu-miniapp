package appsettings

import (
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
	"github.com/cqh6666/caipu-miniapp/backend/internal/credentialcipher"
)

func TestServicesShareInjectedCredentialBox(t *testing.T) {
	t.Parallel()

	sharedBox, err := credentialcipher.New(
		credentialcipher.Key{Version: "current", Secret: "current-secret"},
		[]credentialcipher.Key{{Version: "old", Secret: "old-secret"}},
	)
	if err != nil {
		t.Fatal(err)
	}

	service := NewService(nil, sharedBox, nil, nil)
	provider := NewRuntimeProvider(nil, sharedBox, config.Config{})
	if service.cipherBox != sharedBox || provider.cipherBox != sharedBox {
		t.Fatal("appsettings consumers did not retain the shared credential box")
	}
}
