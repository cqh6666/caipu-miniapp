package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
	"github.com/cqh6666/caipu-miniapp/backend/internal/credentialcipher"
	"github.com/cqh6666/caipu-miniapp/backend/internal/credentialrotate"
	"github.com/cqh6666/caipu-miniapp/backend/internal/db"
)

func main() {
	apply := flag.Bool("apply", false, "write re-encrypted credentials in one transaction")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fatal(err)
	}
	previous, err := credentialcipher.ParsePreviousKeys(cfg.CredentialsPreviousKeys)
	if err != nil {
		fatal(err)
	}
	box, err := credentialcipher.New(credentialcipher.Key{Version: cfg.CredentialsKeyVersion, Secret: cfg.CredentialsSecret}, previous)
	if err != nil {
		fatal(err)
	}
	conn, err := db.Open(cfg, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if err != nil {
		fatal(err)
	}
	defer conn.Close()

	result, err := credentialrotate.Rotate(context.Background(), conn, box, *apply)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("credential rotation apply=%t scanned=%d changed=%d\n", *apply, result.Scanned, result.Changed)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "rotate credentials:", err)
	os.Exit(1)
}
