package tarfs

import (
	"archive/tar"
	"bytes"
	"os"
	"syscall"
)

type File struct {
	h    *tar.Header
	data *bytes.Reader
	open bool
}

func (f *File) Close() error {
	panic("not implemented") // TODO: Implement
}

func (f *File) Read(p []byte) (n int, err error) {
	return f.ReadAt(p, 0)
}

func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	if f.h.Typeflag == tar.TypeDir {
		return 0, syscall.EISDIR
	}
	return f.data.ReadAt(p, off)
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	panic("not implemented") // TODO: Implement
}

func (f *File) Write(p []byte) (n int, err error) { return 0, syscall.EROFS }

func (f *File) WriteAt(p []byte, off int64) (n int, err error) { return 0, syscall.EROFS }

func (f *File) Name() string {
	panic("not implemented") // TODO: Implement
}

func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	panic("not implemented") // TODO: Implement
}

func (f *File) Readdirnames(n int) ([]string, error) {
	panic("not implemented") // TODO: Implement
}

func (f *File) Stat() (os.FileInfo, error) { return f.h.FileInfo(), nil }

func (f *File) Sync() error { return nil }

func (f *File) Truncate(size int64) error { return syscall.EROFS }

func (f *File) WriteString(s string) (ret int, err error) { return 0, syscall.EROFS }
