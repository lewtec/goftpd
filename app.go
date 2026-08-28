package goftpd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"
)

// Config is the knobs the command line fills in before NewApp.
type Config struct {
	Addr string
	Dir  string
}

// App serves Config.Dir over HTTP.
type App struct {
	srv *http.Server
	dir string
}

var _ http.Handler = (*App)(nil)

// NewApp validates cfg and builds a server that is not yet listening.
func NewApp(cfg Config) (*App, error) {
	if cfg.Addr == "" {
		cfg.Addr = ":80"
	}
	if cfg.Dir == "" {
		cfg.Dir = "./"
	}
	if _, err := os.Stat(cfg.Dir); err != nil {
		return nil, fmt.Errorf("pasta de trabalho %q: %w", cfg.Dir, err)
	}

	a := &App{dir: cfg.Dir}
	a.srv = &http.Server{
		Addr:    cfg.Addr,
		Handler: a,
	}
	return a, nil
}

// Run listens until ctx is canceled, then shuts the server down.
// The ListenAndServe goroutine exits after Shutdown or a listen error.
func (a *App) Run(ctx context.Context) error {
	slog.InfoContext(ctx, "iniciando servidor", "dir", a.dir, "addr", a.srv.Addr)
	slog.InfoContext(ctx, "pau na máquina")

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

// ServeHTTP serves a file from the configured directory.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := a.dir + r.RequestURI
	path, err := url.QueryUnescape(path)
	if err != nil {
		slog.WarnContext(r.Context(), "não foi possível parsear a url", "err", err)
	}
	slog.InfoContext(r.Context(), "request",
		"host", r.Host,
		"method", r.Method,
		"url", r.URL.String(),
	)
	http.ServeFile(w, r, path)
	w.Header().Set("Cache-Control", "max-age=5")
}
