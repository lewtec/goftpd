package goftpd

import (
	"cmp"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
	"time"
)

type crumb struct {
	Name string
	Href string
}

type listEntry struct {
	Name  string
	Href  string
	Size  string
	Mtime string
}

func breadcrumbs(urlPath string) []crumb {
	urlPath = strings.Trim(urlPath, "/")
	out := []crumb{{Name: "/", Href: "/"}}
	if urlPath == "" {
		out[0].Href = ""
		return out
	}
	parts := strings.Split(urlPath, "/")
	acc := ""
	for i, p := range parts {
		acc += "/" + p
		c := crumb{Name: p, Href: acc + "/"}
		if i == len(parts)-1 {
			c.Href = ""
		}
		out = append(out, c)
	}
	return out
}

func parentURL(urlPath string) string {
	if urlPath == "/" || urlPath == "" {
		return ""
	}
	p := path.Dir(strings.TrimSuffix(urlPath, "/"))
	if p == "/" || p == "." {
		return "/"
	}
	return p + "/"
}

func sortDirEntries(entries []fs.DirEntry) {
	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		if a.IsDir() != b.IsDir() {
			if a.IsDir() {
				return -1
			}
			return 1
		}
		return cmp.Compare(strings.ToLower(a.Name()), strings.ToLower(b.Name()))
	})
}

func formatSize(n int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	v := float64(n)
	i := 0
	for v >= 1000 && i < len(units)-1 {
		v /= 1000
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d B", n)
	}
	return fmt.Sprintf("%.1f %s", v, units[i])
}

func formatTime(t time.Time) string {
	return t.Local().Format("2006-01-02 15:04")
}
