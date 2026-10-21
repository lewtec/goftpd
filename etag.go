package goftpd

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"sync"
)

// fileStamp is the metadata that lets goftpd reuse a content hash.
// The ETag itself is still the SHA-256 of the bytes.
type fileStamp struct {
	size int64
	mod  int64
	ino  uint64
	dev  uint64
}

func stampOf(fi fs.FileInfo) fileStamp {
	st := fileStamp{size: fi.Size(), mod: fi.ModTime().UnixNano()}
	fillFileID(&st, fi)
	return st
}

type etagEntry struct {
	stamp fileStamp
	etag  string
}

// etagCache remembers a strong ETag per path while fileStamp stays the same.
// The tag is quoted lowercase SHA-256, so it survives a process restart
// even though this map does not.
type etagCache struct {
	mu sync.Mutex
	m  map[string]etagEntry
}

func newETagCache() etagCache {
	return etagCache{m: make(map[string]etagEntry)}
}

// fileETag returns the strong ETag for rs. On a stamp hit it does not read.
// rs is left at the start when err is nil.
func (c *etagCache) fileETag(key string, fi fs.FileInfo, rs io.ReadSeeker) (string, error) {
	st := stampOf(fi)
	c.mu.Lock()
	if e, ok := c.m[key]; ok && e.stamp == st {
		c.mu.Unlock()
		return e.etag, nil
	}
	c.mu.Unlock()

	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	sum := sha256.New()
	if _, err := io.Copy(sum, rs); err != nil {
		return "", err
	}
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	etag := `"` + hex.EncodeToString(sum.Sum(nil)) + `"`
	c.store(key, st, etag)
	return etag, nil
}

// store drops a hash whose stamp is older than the one already cached.
// That happens when the file changes while another request is still reading it.
func (c *etagCache) store(key string, st fileStamp, etag string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cur, ok := c.m[key]; ok && cur.stamp != st && cur.stamp.mod >= st.mod {
		return
	}
	c.m[key] = etagEntry{stamp: st, etag: etag}
}
