// package tarfs implements a read-only in-memory representation of a tar archive
package tarfs

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/afero"
)

var (
	ErrNotImplemented = errors.New("Not implemented")
)

type Fs struct {
	files map[string]*File
}

func New(t *tar.Reader) *Fs {
	fs := &Fs{files: make(map[string]*File)}
	for {
		hdr, err := t.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil
		}

		f := &File{
			h:    hdr,
			open: false,
		}

		var buf bytes.Buffer
		size, err := buf.ReadFrom(t)
		if err != nil {
			panic("tarfs: reading from tar:" + err.Error())
		}

		if size != f.h.Size {
			panic("tarfs: size mismatch")
		}

		f.data = bytes.NewReader(buf.Bytes())

		fs.files[filepath.Clean(hdr.Name)] = f

	}

	return fs
}

func (fs *Fs) Open(name string) (afero.File, error) {
	f, ok := fs.files[name]
	if !ok {
		return nil, &os.PathError{Op: "open", Path: name, Err: syscall.ENOENT}
	}

	f.open = true
	f.data.Seek(0, io.SeekStart)

	return f, nil
}

func (fs *Fs) Name() string { return "tarfs" }

func (fs *Fs) Create(name string) (afero.File, error) {
	return nil, syscall.EROFS
}

func (fs *Fs) Mkdir(name string, perm os.FileMode) error {
	return syscall.EROFS
}

func (fs *Fs) MkdirAll(path string, perm os.FileMode) error {
	return syscall.EROFS
}

func (fs *Fs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	panic("not implemented")
}

func (fs *Fs) Remove(name string) error {
	return syscall.EROFS
}

func (fs *Fs) RemoveAll(path string) error {
	return syscall.EROFS
}

func (fs *Fs) Rename(oldname string, newname string) error {
	return syscall.EROFS
}

func (fs *Fs) Stat(name string) (os.FileInfo, error) {
	panic("not implemented")
}

func (fs *Fs) Chmod(name string, mode os.FileMode) error {
	return syscall.EROFS
}

func (fs *Fs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return syscall.EROFS
}
