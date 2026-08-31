package goftpd

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestApp(t *testing.T, cfg Config) *App {
	t.Helper()
	app, err := NewApp(cfg)
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	return app
}

func doReq(t *testing.T, app *App, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	return doReqLang(t, app, method, target, "")
}

func doReqLang(t *testing.T, app *App, method, target, accept string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, target, nil)
	if accept != "" {
		req.Header.Set("Accept-Language", accept)
	}
	app.ServeHTTP(rec, req)
	return rec
}

func TestNewAppMissingDir(t *testing.T) {
	_, err := NewApp(Config{Addr: ":8080", Dir: filepath.Join(t.TempDir(), "missing")})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("NewApp missing dir: got %v, want os.ErrNotExist", err)
	}
}

func TestNewAppOK(t *testing.T) {
	app := newTestApp(t, Config{Addr: ":8080", Dir: t.TempDir()})
	if app == nil {
		t.Fatal("NewApp returned nil app")
	}
}

func TestServeHTTPFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t, Config{Addr: ":8080", Dir: dir})

	rec := doReq(t, app, http.MethodGet, "/hello.txt")
	if rec.Code != http.StatusOK {
		t.Fatalf("code got %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "hi" {
		t.Fatalf("body got %q, want %q", got, "hi")
	}
	if got := rec.Header().Get("Cache-Control"); got != "max-age=5" {
		t.Fatalf("Cache-Control got %q, want max-age=5", got)
	}
}

func TestListingShowsDotfilesAndIndex(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".hidden"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<p>app</p>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t, Config{Dir: dir})

	rec := doReq(t, app, http.MethodGet, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("code got %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{"hello.txt", ".hidden", "index.html", "sub/"} {
		if !strings.Contains(body, want) {
			t.Fatalf("listing missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "<p>app</p>") {
		t.Fatalf("listing served index.html body")
	}
	if !strings.Contains(body, `<a href="/">root</a>`) {
		t.Fatalf("root crumb is not a link to /:\n%s", body)
	}
	for _, want := range []string{`<th nowrap>Size</th>`, `<th nowrap>Modified</th>`, `<td nowrap>`} {
		if !strings.Contains(body, want) {
			t.Fatalf("listing missing %q:\n%s", want, body)
		}
	}
}

func TestListingDoesNotUseSite404(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "404.html"), []byte("site-404"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t, Config{Dir: dir})

	rec := doReq(t, app, http.MethodGet, "/missing")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("code got %d, want %d", rec.Code, http.StatusNotFound)
	}
	body := rec.Body.String()
	if strings.Contains(body, "site-404") {
		t.Fatalf("non-SPA miss used site 404.html")
	}
	if !strings.Contains(body, "Not found") {
		t.Fatalf("built-in 404 missing: %s", body)
	}
}

func TestDirRedirectTrailingSlash(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t, Config{Dir: dir})

	rec := doReq(t, app, http.MethodGet, "/sub")
	if rec.Code != http.StatusPermanentRedirect {
		t.Fatalf("code got %d, want %d", rec.Code, http.StatusPermanentRedirect)
	}
	if got := rec.Header().Get("Location"); got != "/sub/" {
		t.Fatalf("Location got %q, want /sub/", got)
	}
}

func TestSPANeverLists(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "photo.jpg"), []byte("img"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t, Config{Dir: dir, SPA: true})

	rec := doReq(t, app, http.MethodGet, "/")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("code got %d, want %d", rec.Code, http.StatusNotFound)
	}
	if strings.Contains(rec.Body.String(), "photo.jpg") {
		t.Fatalf("SPA listed files: %s", rec.Body.String())
	}
}

func TestSPADirectoryIndex(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "app", "index.html"), []byte("spa-app"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t, Config{Dir: dir, SPA: true})

	rec := doReq(t, app, http.MethodGet, "/app/")
	if rec.Code != http.StatusOK {
		t.Fatalf("code got %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "spa-app" {
		t.Fatalf("body got %q, want spa-app", got)
	}
}

func TestSPAMissChain(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		files    map[string]string
		target   string
		wantCode int
		wantBody string
	}{
		{
			name:     "site 404 wins",
			files:    map[string]string{"404.html": "site-404", "index.html": "site-index"},
			target:   "/nope",
			wantCode: http.StatusNotFound,
			wantBody: "site-404",
		},
		{
			name:     "site index when no 404",
			files:    map[string]string{"index.html": "site-index"},
			target:   "/nope",
			wantCode: http.StatusOK,
			wantBody: "site-index",
		},
		{
			name:     "built-in when neither",
			files:    map[string]string{"other.txt": "x"},
			target:   "/nope",
			wantCode: http.StatusNotFound,
			wantBody: "Not found",
		},
		{
			name: "no walk up",
			files: map[string]string{
				"app/index.html": "nested",
			},
			target:   "/app/user/42",
			wantCode: http.StatusNotFound,
			wantBody: "Not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for name, body := range tt.files {
				p := filepath.Join(dir, name)
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			app := newTestApp(t, Config{Dir: dir, SPA: true})
			rec := doReq(t, app, http.MethodGet, tt.target)
			if rec.Code != tt.wantCode {
				t.Fatalf("code got %d, want %d", rec.Code, tt.wantCode)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Fatalf("body %q does not contain %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestReservedAssets(t *testing.T) {
	app := newTestApp(t, Config{Dir: t.TempDir(), SPA: true})

	rec := doReq(t, app, http.MethodGet, "/__goftpd__/sakura.css")
	if rec.Code != http.StatusOK {
		t.Fatalf("sakura.css code got %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "Sakura.css") {
		t.Fatalf("sakura.css body unexpected")
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/css") {
		t.Fatalf("Content-Type got %q", ct)
	}

	rec = doReq(t, app, http.MethodGet, "/__goftpd__/missing.css")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing asset code got %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	app := newTestApp(t, Config{Dir: t.TempDir()})
	rec := doReq(t, app, http.MethodPost, "/")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("code got %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestQueryStringNotInPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t, Config{Dir: dir})
	rec := doReq(t, app, http.MethodGet, "/hello.txt?download=1")
	if rec.Code != http.StatusOK {
		t.Fatalf("code got %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "hi" {
		t.Fatalf("body got %q, want hi", got)
	}
}

func TestHeadListing(t *testing.T) {
	app := newTestApp(t, Config{Dir: t.TempDir()})
	rec := doReq(t, app, http.MethodHead, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("code got %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("HEAD body not empty: %q", rec.Body.String())
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

func TestBreadcrumbs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path string
		want []crumb
	}{
		{
			path: "/",
			want: []crumb{{Name: "root", Href: "/"}},
		},
		{
			path: "/docs/",
			want: []crumb{
				{Name: "root", Href: "/"},
				{Name: "docs", Href: "/docs/"},
			},
		},
		{
			path: "/docs/archive/2024/jan/",
			want: []crumb{
				{Name: "root", Href: "/"},
				{Name: "docs", Href: "/docs/"},
				{Name: "archive", Href: "/docs/archive/"},
				{Name: "2024", Href: "/docs/archive/2024/"},
				{Name: "jan", Href: "/docs/archive/2024/jan/"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			got := breadcrumbs(tt.path, "root")
			if len(got) != len(tt.want) {
				t.Fatalf("breadcrumbs(%q) len=%d, want %d: %#v", tt.path, len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("breadcrumbs(%q)[%d] = %+v, want %+v", tt.path, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestListingPortuguese(t *testing.T) {
	app := newTestApp(t, Config{Dir: t.TempDir()})
	rec := doReqLang(t, app, http.MethodGet, "/", "pt")
	if rec.Code != http.StatusOK {
		t.Fatalf("code got %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{`lang="pt"`, "Índice de /", "raiz", "Nome", "Tamanho", "Modificado"} {
		if !strings.Contains(body, want) {
			t.Fatalf("pt listing missing %q:\n%s", want, body)
		}
	}
	if got := rec.Header().Get("Content-Language"); got != "pt" {
		t.Fatalf("Content-Language got %q, want pt", got)
	}
}

func TestNotFoundPortuguese(t *testing.T) {
	app := newTestApp(t, Config{Dir: t.TempDir()})
	rec := doReqLang(t, app, http.MethodGet, "/missing", "pt-BR,pt;q=0.9")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("code got %d, want %d", rec.Code, http.StatusNotFound)
	}
	body := rec.Body.String()
	for _, want := range []string{"Não encontrado", "Nenhum arquivo em", "Voltar para /"} {
		if !strings.Contains(body, want) {
			t.Fatalf("pt 404 missing %q:\n%s", want, body)
		}
	}
}

func TestLangQueryIgnored(t *testing.T) {
	app := newTestApp(t, Config{Dir: t.TempDir()})
	rec := doReqLang(t, app, http.MethodGet, "/?lang=pt", "en")
	if rec.Code != http.StatusOK {
		t.Fatalf("code got %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Index of /") || strings.Contains(body, "Índice") {
		t.Fatalf("query lang must not override Accept-Language:\n%s", body)
	}
}

func TestMethodNotAllowedPortuguese(t *testing.T) {
	app := newTestApp(t, Config{Dir: t.TempDir()})
	rec := doReqLang(t, app, http.MethodPost, "/", "pt")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("code got %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if !strings.Contains(rec.Body.String(), "método não permitido") {
		t.Fatalf("pt 405 body: %q", rec.Body.String())
	}
}

func TestLocalizeMissingID(t *testing.T) {
	t.Parallel()
	got := localize(Localizer("en"), "DoesNotExist", nil)
	if got != "DoesNotExist" {
		t.Fatalf("missing id got %q, want the id back", got)
	}
}

func TestFormatSize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		n    int64
		want string
	}{
		{0, "0 B"},
		{999, "999 B"},
		{1000, "1.0 KB"},
		{1500, "1.5 KB"},
		{1_000_000, "1.0 MB"},
	}
	for _, tt := range tests {
		if got := formatSize(tt.n); got != tt.want {
			t.Fatalf("formatSize(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}
