package goftpd

import (
	"cmp"
	"context"
	"errors"
	"fmt"
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

// App serves Config.Dir over HTTP.
type App struct {
	srv  *http.Server
	dir  string
	root *os.Root
	spa  bool
}

var _ http.Handler = (*App)(nil)

// NewApp validates cfg and builds a server that is not yet listening.
func NewApp(cfg Config) (*App, error) {
	cfg.Addr = cmp.Or(cfg.Addr, ":8080")
	cfg.Dir = cmp.Or(cfg.Dir, "./")
	abs, err := filepath.Abs(cfg.Dir)
	if err != nil {
		return nil, fmt.Errorf("served directory %q: %w", cfg.Dir, err)
	}
	root, err := os.OpenRoot(abs)
	if err != nil {
		return nil, fmt.Errorf("served directory %q: %w", cfg.Dir, err)
	}

	a := &App{dir: abs, root: root, spa: cfg.SPA}
	a.srv = &http.Server{
		Addr:              cfg.Addr,
		Handler:           a,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return a, nil
}

// Close releases the served directory handle.
func (a *App) Close() error {
	if a.root == nil {
		return nil
	}
	return a.root.Close()
}

// Run listens until ctx is canceled, then shuts the server down.
// The ListenAndServe goroutine exits after Shutdown or a listen error.
func (a *App) Run(ctx context.Context) error {
	defer a.Close()
	slog.InfoContext(ctx, "starting server", "dir", a.dir, "addr", a.srv.Addr, "spa", a.spa)

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
