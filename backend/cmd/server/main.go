package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/app"
	"github.com/cqh6666/caipu-miniapp/backend/internal/buildinfo"
	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
)

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "run migrations and exit")
	checkConfig := flag.Bool("check-config", false, "validate configuration and exit without opening the database")
	versionOnly := flag.Bool("version", false, "print build identity as JSON and exit")
	flag.Parse()
	if *versionOnly {
		if err := json.NewEncoder(os.Stdout).Encode(buildinfo.Current()); err != nil {
			log.Fatalf("encode build identity: %v", err)
		}
		return
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if *checkConfig {
		log.Printf("configuration valid: env=%s sources=%s", cfg.AppEnv, cfg.ConfigSourceSummary)
		return
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("create app: %v", err)
	}

	if *migrateOnly {
		application.Logger.Info("migrations completed; exiting because migrate-only mode is enabled")
		return
	}

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- application.Start()
	}()

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErrCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			application.Logger.Error("server exited unexpectedly", "error", err)
			os.Exit(1)
		}
	case <-signalCtx.Done():
		application.Logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := application.Shutdown(shutdownCtx); err != nil {
		application.Logger.Error("shutdown failed", "error", err)
		os.Exit(1)
	}
}
