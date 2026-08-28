package goftpd

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewAppMissingDir(t *testing.T) {
	_, err := NewApp(Config{Addr: ":8080", Dir: filepath.Join(t.TempDir(), "missing")})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("NewApp missing dir: got %v, want os.ErrNotExist", err)
	}
}

func TestNewAppOK(t *testing.T) {
	app, err := NewApp(Config{Addr: ":8080", Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	if app == nil {
		t.Fatal("NewApp returned nil app")
	}
}

func TestServeHTTP(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	app, err := NewApp(Config{Addr: ":8080", Dir: dir})
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/hello.txt", nil)
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("code got %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "hi" {
		t.Fatalf("body got %q, want %q", got, "hi")
	}
}

func TestRunStopsOnCancel(t *testing.T) {
	app, err := NewApp(Config{Addr: ":0", Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	if err := app.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}
}
