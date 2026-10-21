package goftpd

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

var etagRe = regexp.MustCompile(`^"[0-9a-f]{64}"$`)

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

func TestETagFollowsBytesNotMtime(t *testing.T) {
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
	sum := sha256.Sum256(body)
	etag := `"` + hex.EncodeToString(sum[:]) + `"`
	if got := rec.Header().Get("ETag"); got != etag {
		t.Fatalf("ETag got %q, want %q", got, etag)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control got %q, want no-cache", got)
	}

	future := time.Now().Add(2 * time.Hour)
	if err := os.Chtimes(p, future, future); err != nil {
		t.Fatal(err)
	}
	rec = doReqHdr(t, app, http.MethodGet, "/clip.bin", map[string]string{"If-None-Match": etag})
	if rec.Code != http.StatusNotModified {
		t.Fatalf("mtime-only code got %d, want 304", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("304 body = %q", rec.Body.String())
	}
	if got := rec.Header().Get("ETag"); got != etag {
		t.Fatalf("304 ETag got %q, want %q", got, etag)
	}

	changed := []byte("0123456789abcdeX")
	if err := os.WriteFile(p, changed, 0o644); err != nil {
		t.Fatal(err)
	}
	rec = doReqHdr(t, app, http.MethodGet, "/clip.bin", map[string]string{"If-None-Match": etag})
	if rec.Code != http.StatusOK {
		t.Fatalf("changed code got %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != string(changed) {
		t.Fatalf("body got %q", got)
	}
	if got := rec.Header().Get("ETag"); got == etag || !etagRe.MatchString(got) {
		t.Fatalf("changed ETag got %q", got)
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
		"If-Range": `"0000000000000000000000000000000000000000000000000000000000000000"`,
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

	rec = doReqHdr(t, app, http.MethodGet, "/clip.bin", map[string]string{
		"If-Match": `"0000000000000000000000000000000000000000000000000000000000000000"`,
	})
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
	etag := rec.Header().Get("ETag")
	if !etagRe.MatchString(etag) {
		t.Fatalf("ETag got %q", etag)
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

type stubInfo struct {
	size int64
	mod  time.Time
}

func (s stubInfo) Name() string       { return "f" }
func (s stubInfo) Size() int64        { return s.size }
func (s stubInfo) Mode() fs.FileMode  { return 0o644 }
func (s stubInfo) ModTime() time.Time { return s.mod }
func (s stubInfo) IsDir() bool        { return false }
func (s stubInfo) Sys() any           { return nil }

type memFile struct {
	data  []byte
	off   int64
	reads int
}

func (m *memFile) Read(p []byte) (int, error) {
	m.reads++
	if m.off >= int64(len(m.data)) {
		return 0, io.EOF
	}
	n := copy(p, m.data[m.off:])
	m.off += int64(n)
	return n, nil
}

func (m *memFile) Seek(off int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		m.off = off
	case io.SeekCurrent:
		m.off += off
	case io.SeekEnd:
		m.off = int64(len(m.data)) + off
	default:
		return 0, fs.ErrInvalid
	}
	return m.off, nil
}

func TestETagCacheSkipsReread(t *testing.T) {
	data := []byte("same-bytes")
	info := stubInfo{size: int64(len(data)), mod: time.Unix(10, 0)}
	cache := newETagCache()
	first := &memFile{data: data}
	etag, err := cache.fileETag("file:a", info, first)
	if err != nil {
		t.Fatal(err)
	}
	if first.reads == 0 {
		t.Fatal("first call did not read")
	}
	second := &memFile{data: data}
	got, err := cache.fileETag("file:a", info, second)
	if err != nil {
		t.Fatal(err)
	}
	if got != etag {
		t.Fatalf("cached etag %q, want %q", got, etag)
	}
	if second.reads != 0 {
		t.Fatalf("cache hit read the file %d times", second.reads)
	}

	info.mod = info.mod.Add(time.Second)
	third := &memFile{data: data}
	got, err = cache.fileETag("file:a", info, third)
	if err != nil {
		t.Fatal(err)
	}
	if got != etag {
		t.Fatalf("same bytes etag %q, want %q", got, etag)
	}
	if third.reads == 0 {
		t.Fatal("stamp change did not rehash")
	}
}
