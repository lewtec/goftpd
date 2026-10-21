package goftpd

import (
	"os"
	"strconv"
	"sync"
	"time"
)

// mtimeETag is the strong validator for one file. A new modification time
// misses. The same modification time matches, even when the bytes differ.
func mtimeETag(mod time.Time) string {
	return `"` + strconv.FormatInt(mod.UnixNano(), 10) + `"`
}

// validatorModTime is the file modification time. embed.FS reports the zero
// time, so those files use the server binary's modification time, captured
// once. The running process keeps that tag until restart.
func validatorModTime(mod time.Time) time.Time {
	if !mod.IsZero() {
		return mod
	}
	return executableModTime()
}

var (
	exeModOnce sync.Once
	exeMod     time.Time
)

func executableModTime() time.Time {
	exeModOnce.Do(func() {
		exeMod = time.Now()
		exe, err := os.Executable()
		if err != nil {
			return
		}
		fi, err := os.Stat(exe)
		if err != nil {
			return
		}
		if mod := fi.ModTime(); !mod.IsZero() {
			exeMod = mod
		}
	})
	return exeMod
}
