package app

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

type blockingShutdownWorker struct {
	stopStarted chan struct{}
	release     chan struct{}
}

func (w *blockingShutdownWorker) Start(context.Context) error { return nil }
func (w *blockingShutdownWorker) Stop(context.Context) error {
	close(w.stopStarted)
	<-w.release
	return nil
}

func TestAppShutdownDeadlineBoundsNonCooperativeWorkerAndStillClosesDB(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Ping(); err != nil {
		t.Fatal(err)
	}
	worker := &blockingShutdownWorker{
		stopStarted: make(chan struct{}),
		release:     make(chan struct{}),
	}
	application := &App{
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:      database,
		workers: []backgroundWorker{worker},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	startedAt := time.Now()
	err = application.Shutdown(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown() error=%v, want deadline exceeded", err)
	}
	if elapsed := time.Since(startedAt); elapsed > 500*time.Millisecond {
		t.Fatalf("Shutdown() exceeded bounded wait: %s", elapsed)
	}
	select {
	case <-worker.stopStarted:
	default:
		t.Fatal("worker stop was not attempted")
	}
	if err := database.Ping(); err == nil {
		t.Fatal("database remained open after bounded shutdown")
	}
	close(worker.release)
}

type startErrorWorker struct {
	err error
}

func (w *startErrorWorker) Start(context.Context) error { return w.err }
func (w *startErrorWorker) Stop(context.Context) error  { return nil }

func TestAppStartReturnsWorkerValidationErrorBeforeServing(t *testing.T) {
	want := errors.New("invalid worker interval")
	application := &App{workers: []backgroundWorker{&startErrorWorker{err: want}}}
	err := application.Start()
	if !errors.Is(err, want) {
		t.Fatalf("Start() error=%v, want=%v", err, want)
	}
}
