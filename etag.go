package goftpd

import (
	"strconv"
	"time"
)

// mtimeETag is the strong validator for one file. A new modification time
// misses. The same modification time matches, even when the bytes differ.
func mtimeETag(mod time.Time) string {
	return `"` + strconv.FormatInt(mod.UnixNano(), 10) + `"`
}
