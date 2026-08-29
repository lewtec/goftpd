package goftpd

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Config is the knobs the command line fills in before NewApp.
type Config struct {
	Addr string
	Dir  string
	SPA  bool
}

// App serves an fs.FS over HTTP.
type App struct {
	srv  *http.Server
	fsys fs.FS
	spa  bool
}

var _ http.Handler = (*App)(nil)

// NewApp validates cfg and builds a server that is not yet listening.
func NewApp(cfg Config) (*App, error) {
	cfg.Addr = cmp.Or(cfg.Addr, ":8080")
	fsys, err := dirFS(cfg.Dir)
	if err != nil {
		return nil, err
	}

	a := &App{fsys: fsys, spa: cfg.SPA}
	a.srv = &http.Server{
		Addr:              cfg.Addr,
		Handler:           a,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return a, nil
}

func dirFS(dir string) (fs.FS, error) {
	dir = cmp.Or(dir, "./")
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("served directory %q: %w", dir, err)
	}
	if _, err := os.Stat(abs); err != nil {
		return nil, fmt.Errorf("served directory %q: %w", dir, err)
	}
	return os.DirFS(abs), nil
}

// Run listens until ctx is canceled, then shuts the server down.
// The ListenAndServe goroutine exits after Shutdown or a listen error.
func (a *App) Run(ctx context.Context) error {
	slog.InfoContext(ctx, "starting server", "addr", a.srv.Addr, "spa", a.spa)

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := a.srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		err := <-errCh
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
