package goftpd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

var etagRe = regexp.MustCompile(`^"-?[0-9]+"$`)

func doReqHdr(t *testing.T, app *App, method, target string, hdr map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, target, nil)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	app.ServeHTTP(rec, req)
	return rec
}

func TestMtimeETagUnstales(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "clip.bin")
	body := []byte("0123456789abcdef")
	if err := os.WriteFile(p, body, 0o644); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t, Config{Dir: dir})

	rec := doReq(t, app, http.MethodGet, "/clip.bin")
	if rec.Code != http.StatusOK {
		t.Fatalf("code got %d, want 200", rec.Code)
	}
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	etag := mtimeETag(fi.ModTime())
	if got := rec.Header().Get("ETag"); got != etag {
		t.Fatalf("ETag got %q, want %q", got, etag)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control got %q, want no-cache", got)
	}

	rec = doReqHdr(t, app, http.MethodGet, "/clip.bin", map[string]string{"If-None-Match": etag})
	if rec.Code != http.StatusNotModified {
		t.Fatalf("same mtime code got %d, want 304", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("304 body = %q", rec.Body.String())
	}

	changed := []byte("totally-different")
	if err := os.WriteFile(p, changed, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, fi.ModTime(), fi.ModTime()); err != nil {
		t.Fatal(err)
	}
	rec = doReqHdr(t, app, http.MethodGet, "/clip.bin", map[string]string{"If-None-Match": etag})
	if rec.Code != http.StatusNotModified {
		t.Fatalf("same mtime after rewrite code got %d, want 304", rec.Code)
	}

	future := fi.ModTime().Add(2 * time.Hour)
	if err := os.Chtimes(p, future, future); err != nil {
		t.Fatal(err)
	}
	rec = doReqHdr(t, app, http.MethodGet, "/clip.bin", map[string]string{"If-None-Match": etag})
	if rec.Code != http.StatusOK {
		t.Fatalf("new mtime code got %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != string(changed) {
		t.Fatalf("body got %q", got)
	}
	fi, err = os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := rec.Header().Get("ETag"); got == etag || got != mtimeETag(fi.ModTime()) {
		t.Fatalf("new ETag got %q, old %q", got, etag)
	}
}

func TestByteRangeFromMiddle(t *testing.T) {
	dir := t.TempDir()
	body := []byte("0123456789")
	if err := os.WriteFile(filepath.Join(dir, "clip.bin"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t, Config{Dir: dir})

	rec := doReqHdr(t, app, http.MethodGet, "/clip.bin", map[string]string{"Range": "bytes=4-7"})
	if rec.Code != http.StatusPartialContent {
		t.Fatalf("code got %d, want 206", rec.Code)
	}
	if got := rec.Body.String(); got != "4567" {
		t.Fatalf("slice got %q, want 4567", got)
	}
	if got := rec.Header().Get("Content-Range"); got != "bytes 4-7/10" {
		t.Fatalf("Content-Range got %q", got)
	}
	if got := rec.Header().Get("Content-Length"); got != "4" {
		t.Fatalf("Content-Length got %q", got)
	}
	etag := rec.Header().Get("ETag")
	if !etagRe.MatchString(etag) {
		t.Fatalf("ETag got %q", etag)
	}

	rec = doReqHdr(t, app, http.MethodGet, "/clip.bin", map[string]string{
		"Range":    "bytes=8-",
		"If-Range": etag,
	})
	if rec.Code != http.StatusPartialContent {
		t.Fatalf("If-Range code got %d, want 206", rec.Code)
	}
	if got := rec.Body.String(); got != "89" {
		t.Fatalf("open range got %q, want 89", got)
	}
	if got := rec.Header().Get("Content-Range"); got != "bytes 8-9/10" {
		t.Fatalf("Content-Range got %q", got)
	}

	rec = doReqHdr(t, app, http.MethodGet, "/clip.bin", map[string]string{
		"Range":    "bytes=4-7",
		"If-Range": `"0"`,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("stale If-Range code got %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != string(body) {
		t.Fatalf("stale If-Range body got %q", got)
	}

	rec = doReqHdr(t, app, http.MethodHead, "/clip.bin", map[string]string{"Range": "bytes=4-7"})
	if rec.Code != http.StatusPartialContent {
		t.Fatalf("HEAD code got %d, want 206", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("HEAD body = %q", rec.Body.String())
	}
	if got := rec.Header().Get("Content-Range"); got != "bytes 4-7/10" {
		t.Fatalf("HEAD Content-Range got %q", got)
	}

	rec = doReqHdr(t, app, http.MethodGet, "/clip.bin", map[string]string{"Range": "bytes=20-30"})
	if rec.Code != http.StatusRequestedRangeNotSatisfiable {
		t.Fatalf("code got %d, want 416", rec.Code)
	}

	rec = doReqHdr(t, app, http.MethodGet, "/clip.bin", map[string]string{"If-Match": `"0"`})
	if rec.Code != http.StatusPreconditionFailed {
		t.Fatalf("If-Match code got %d, want 412", rec.Code)
	}
}

func TestAssetRevalidates(t *testing.T) {
	app := newTestApp(t, Config{Dir: t.TempDir()})
	rec := doReq(t, app, http.MethodGet, "/__goftpd__/sakura.css")
	if rec.Code != http.StatusOK {
		t.Fatalf("code got %d, want 200", rec.Code)
	}
	etag := mtimeETag(executableModTime())
	if got := rec.Header().Get("ETag"); got != etag {
		t.Fatalf("ETag got %q, want %q", got, etag)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control got %q, want no-cache", got)
	}
	rec = doReqHdr(t, app, http.MethodGet, "/__goftpd__/sakura.css", map[string]string{"If-None-Match": etag})
	if rec.Code != http.StatusNotModified {
		t.Fatalf("code got %d, want 304", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("304 body = %q", rec.Body.String())
	}

	rec = doReq(t, app, http.MethodGet, "/__goftpd__/sakura-dark.css")
	if got := rec.Header().Get("ETag"); got != etag {
		t.Fatalf("dark css ETag got %q, want the binary mtime %q", got, etag)
	}
}

func TestSite404IgnoresRange(t *testing.T) {
	dir := t.TempDir()
	body := "0123456789site-404"
	if err := os.WriteFile(filepath.Join(dir, "404.html"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t, Config{Dir: dir, SPA: true})
	rec := doReqHdr(t, app, http.MethodGet, "/missing", map[string]string{"Range": "bytes=0-3"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("code got %d, want 404", rec.Code)
	}
	if got := rec.Body.String(); got != body {
		t.Fatalf("site 404 body got %q, want the whole file", got)
	}
}

func TestListingKeepsShortCache(t *testing.T) {
	app := newTestApp(t, Config{Dir: t.TempDir()})
	rec := doReq(t, app, http.MethodGet, "/")
	if got := rec.Header().Get("Cache-Control"); got != "max-age=5" {
		t.Fatalf("listing Cache-Control got %q, want max-age=5", got)
	}
	if got := rec.Header().Get("ETag"); got != "" {
		t.Fatalf("listing ETag got %q", got)
	}
}
