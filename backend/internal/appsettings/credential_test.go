package appsettings

import (
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/credentialcipher"
)

func testCredentialBox(t testing.TB, secret string) *credentialcipher.Box {
	t.Helper()
	box, err := credentialcipher.New(credentialcipher.Key{Version: "v1", Secret: secret}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return box
}
