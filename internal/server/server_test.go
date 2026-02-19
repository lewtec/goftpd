package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestDirExists(t *testing.T) {
	// Create a temp dir
	tmpDir, err := os.MkdirTemp("", "goftpd_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test existing dir
	if !dirExists(tmpDir) {
		t.Errorf("dirExists should return true for %s", tmpDir)
	}

	// Test non-existing dir
	if dirExists(tmpDir + "/nonexistent") {
		t.Errorf("dirExists should return false for nonexistent dir")
	}
}

func TestLoggingMiddleware(t *testing.T) {
	// Mock handler
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	handler := loggingMiddleware(next)

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Check header
	if w.Header().Get("Cache-Control") != "max-age=5" {
		t.Errorf("Expected Cache-Control: max-age=5, got %s", w.Header().Get("Cache-Control"))
	}

	// Check body
	if w.Body.String() != "ok" {
		t.Errorf("Expected body ok, got %s", w.Body.String())
	}
}
