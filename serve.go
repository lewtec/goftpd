package goftpd

import (
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
)

const (
	reservedPrefix = "/__goftpd__"
	cacheControl   = "max-age=5"
)

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slog.InfoContext(r.Context(), "request",
		"host", r.Host,
		"method", r.Method,
		"url", r.URL.String(),
	)
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	urlPath := cleanURLPath(r.URL.Path)
	if isReserved(urlPath) {
		a.serveAsset(w, r, urlPath)
		return
	}

	rel, err := relFromURL(urlPath)
	if err != nil {
		a.writeNotFound(w, r)
		return
	}

	fi, err := a.root.Stat(rel)
	if err != nil {
		a.serveMiss(w, r)
		return
	}
	if fi.IsDir() {
		if !strings.HasSuffix(urlPath, "/") {
			u := *r.URL
			u.Path = urlPath + "/"
			http.Redirect(w, r, u.String(), http.StatusPermanentRedirect)
			return
		}
		a.serveDir(w, r, rel, urlPath)
		return
	}
	a.serveDiskFile(w, r, rel, http.StatusOK)
}

func (a *App) serveDir(w http.ResponseWriter, r *http.Request, rel, urlPath string) {
	if a.spa {
		idx := "index.html"
		if rel != "." {
			idx = path.Join(rel, "index.html")
		}
		if fileExists(a.root, idx) {
			a.serveDiskFile(w, r, idx, http.StatusOK)
			return
		}
		a.serveMiss(w, r)
		return
	}
	a.writeListing(w, r, rel, urlPath)
}

func (a *App) serveMiss(w http.ResponseWriter, r *http.Request) {
	if !a.spa {
		a.writeNotFound(w, r)
		return
	}
	if fileExists(a.root, "404.html") {
		a.serveDiskFile(w, r, "404.html", http.StatusNotFound)
		return
	}
	if fileExists(a.root, "index.html") {
		a.serveDiskFile(w, r, "index.html", http.StatusOK)
		return
	}
	a.writeNotFound(w, r)
}

func (a *App) writeListing(w http.ResponseWriter, r *http.Request, rel, urlPath string) {
	entries, err := fs.ReadDir(a.root.FS(), rel)
	if err != nil {
		slog.WarnContext(r.Context(), "readdir", "path", rel, "err", err)
		a.writeNotFound(w, r)
		return
	}
	sortDirEntries(entries)
	rows := make([]listEntry, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		href := path.Join(urlPath, url.PathEscape(name))
		info, err := e.Info()
		if err != nil {
			continue
		}
		row := listEntry{
			Name:  name,
			Href:  href,
			Mtime: formatTime(info.ModTime()),
		}
		if e.IsDir() {
			row.Name = name + "/"
			row.Href = href + "/"
		} else {
			row.Size = formatSize(info.Size())
		}
		rows = append(rows, row)
	}
	title := "Index of " + urlPath
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControl)
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	if err := listingPage(title, breadcrumbs(urlPath), parentURL(urlPath), rows).Render(r.Context(), w); err != nil {
		slog.WarnContext(r.Context(), "render listing", "err", err)
	}
}

func (a *App) writeNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControl)
	w.WriteHeader(http.StatusNotFound)
	if r.Method == http.MethodHead {
		return
	}
	if err := notFoundPage(r.URL.Path).Render(r.Context(), w); err != nil {
		slog.WarnContext(r.Context(), "render not found", "err", err)
	}
}

func (a *App) serveDiskFile(w http.ResponseWriter, r *http.Request, rel string, status int) {
	f, err := a.root.Open(rel)
	if err != nil {
		a.writeNotFound(w, r)
		return
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		a.writeNotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", cacheControl)
	if status == http.StatusOK {
		http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
		return
	}
	ctype := mime.TypeByExtension(path.Ext(rel))
	if ctype == "" {
		buf := make([]byte, 512)
		n, err := io.ReadFull(f, buf)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			a.writeNotFound(w, r)
			return
		}
		ctype = http.DetectContentType(buf[:n])
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			a.writeNotFound(w, r)
			return
		}
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
	w.WriteHeader(status)
	if r.Method == http.MethodHead {
		return
	}
	if _, err := io.Copy(w, f); err != nil {
		slog.WarnContext(r.Context(), "copy file", "path", rel, "err", err)
	}
}

func (a *App) serveAsset(w http.ResponseWriter, r *http.Request, urlPath string) {
	name, ok := strings.CutPrefix(urlPath, reservedPrefix+"/")
	if !ok || name == "" {
		a.writeNotFound(w, r)
		return
	}
	name = path.Clean(name)
	if name == "." || strings.HasPrefix(name, "..") {
		a.writeNotFound(w, r)
		return
	}
	f, err := assetsFS.Open("assets/" + name)
	if err != nil {
		a.writeNotFound(w, r)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		a.writeNotFound(w, r)
		return
	}
	rs, ok := f.(io.ReadSeeker)
	if !ok {
		a.writeNotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", cacheControl)
	http.ServeContent(w, r, info.Name(), info.ModTime(), rs)
}

func fileExists(root *os.Root, rel string) bool {
	fi, err := root.Stat(rel)
	return err == nil && !fi.IsDir()
}

func isReserved(urlPath string) bool {
	return urlPath == reservedPrefix || strings.HasPrefix(urlPath, reservedPrefix+"/")
}

func cleanURLPath(p string) string {
	if p == "" {
		return "/"
	}
	cleaned := path.Clean(p)
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	if cleaned != "/" && strings.HasSuffix(p, "/") {
		cleaned += "/"
	}
	return cleaned
}

func relFromURL(urlPath string) (string, error) {
	p := strings.TrimSuffix(urlPath, "/")
	if p == "" || p == "/" {
		return ".", nil
	}
	rel := strings.TrimPrefix(p, "/")
	if rel == "" || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", fs.ErrNotExist
	}
	return rel, nil
}
