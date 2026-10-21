//go:build !unix

package goftpd

import "io/fs"

func fillFileID(_ *fileStamp, _ fs.FileInfo) {}
