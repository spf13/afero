package tarfs

import (
	"archive/tar"
	"os"
	"syscall"
)

type File struct {
	h      *tar.Header
	data   []byte
	at     int64
	closed bool
}

func (f *File) Close() error {
	panic("not implemented") // TODO: Implement
}

func (f *File) Read(p []byte) (n int, err error) {
	if f.h.Typeflag == tar.TypeDir {
		return 0, syscall.EISDIR
	}

	return 0, nil
}

func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	panic("not implemented") // TODO: Implement
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	panic("not implemented") // TODO: Implement
}

func (f *File) Write(p []byte) (n int, err error) {
	panic("not implemented") // TODO: Implement
}

func (f *File) WriteAt(p []byte, off int64) (n int, err error) {
	panic("not implemented") // TODO: Implement
}

func (f *File) Name() string {
	panic("not implemented") // TODO: Implement
}

func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	panic("not implemented") // TODO: Implement
}

func (f *File) Readdirnames(n int) ([]string, error) {
	panic("not implemented") // TODO: Implement
}

func (f *File) Stat() (os.FileInfo, error) {
	panic("not implemented") // TODO: Implement
}

func (f *File) Sync() error {
	panic("not implemented") // TODO: Implement
}

func (f *File) Truncate(size int64) error {
	panic("not implemented") // TODO: Implement
}

func (f *File) WriteString(s string) (ret int, err error) {
	panic("not implemented") // TODO: Implement
}
