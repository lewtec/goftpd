//go:build unix

package goftpd

import (
	"io/fs"
	"syscall"
)

func fillFileID(st *fileStamp, fi fs.FileInfo) {
	sys, ok := fi.Sys().(*syscall.Stat_t)
	if !ok || sys == nil {
		return
	}
	st.ino = uint64(sys.Ino)
	st.dev = uint64(sys.Dev)
}
