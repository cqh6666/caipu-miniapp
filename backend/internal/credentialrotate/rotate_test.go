package credentialrotate

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/credentialcipher"
	_ "modernc.org/sqlite"
)

func TestRotateUpdatesAllCredentialTablesAndSupportsRollbackKeyring(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`
CREATE TABLE app_bilibili_settings (id INTEGER PRIMARY KEY, sessdata_ciphertext TEXT NOT NULL DEFAULT '');
CREATE TABLE app_runtime_settings (key TEXT PRIMARY KEY, value_ciphertext TEXT NOT NULL DEFAULT '');
CREATE TABLE ai_route_providers (id TEXT PRIMARY KEY, api_key_ciphertext TEXT NOT NULL DEFAULT '');
`); err != nil {
		t.Fatal(err)
	}
	oldBox, _ := credentialcipher.New(credentialcipher.Key{Version: "old", Secret: "old-secret"}, nil)
	oldValues := []string{"bilibili", "runtime", "provider"}
	ciphertexts := make([]string, len(oldValues))
	for index, value := range oldValues {
		ciphertexts[index], err = oldBox.Encrypt(value)
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO app_bilibili_settings VALUES (1, ?); INSERT INTO app_runtime_settings VALUES ('a', ?); INSERT INTO ai_route_providers VALUES ('p', ?);`, ciphertexts[0], ciphertexts[1], ciphertexts[2]); err != nil {
		t.Fatal(err)
	}
	newBox, _ := credentialcipher.New(credentialcipher.Key{Version: "new", Secret: "new-secret"}, []credentialcipher.Key{{Version: "old", Secret: "old-secret"}})
	result, err := Rotate(context.Background(), db, newBox, true)
	if err != nil || result.Scanned != 3 || result.Changed != 3 {
		t.Fatalf("Rotate result=%#v error=%v", result, err)
	}
	for _, query := range []string{
		"SELECT sessdata_ciphertext FROM app_bilibili_settings",
		"SELECT value_ciphertext FROM app_runtime_settings",
		"SELECT api_key_ciphertext FROM ai_route_providers",
	} {
		var ciphertext string
		if err := db.QueryRow(query).Scan(&ciphertext); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(ciphertext, "enc:v1:new:") {
			t.Fatalf("rotated ciphertext=%q", ciphertext)
		}
	}
	rollbackBox, _ := credentialcipher.New(credentialcipher.Key{Version: "old", Secret: "old-secret"}, []credentialcipher.Key{{Version: "new", Secret: "new-secret"}})
	result, err = Rotate(context.Background(), db, rollbackBox, true)
	if err != nil || result.Changed != 3 {
		t.Fatalf("rollback result=%#v error=%v", result, err)
	}
}
