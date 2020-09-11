// package tarfs implements a read-only in-memory representation of a tar archive
package tarfs

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/afero"
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
			h: hdr,
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

		name := filepath.Clean(hdr.Name)
		fs.files[name] = f
		f.h.Name = name

	}

	return fs
}

func (fs *Fs) Open(name string) (afero.File, error) {
	f, ok := fs.files[name]
	if !ok {
		return nil, &os.PathError{Op: "open", Path: name, Err: syscall.ENOENT}
	}

	// preserve the original data, and return a copy instead
	cp := &File{h: f.h, data: f.data}

	return cp, nil
}

func (fs *Fs) Name() string { return "tarfs" }

func (fs *Fs) Create(name string) (afero.File, error) { return nil, syscall.EROFS }

func (fs *Fs) Mkdir(name string, perm os.FileMode) error { return syscall.EROFS }

func (fs *Fs) MkdirAll(path string, perm os.FileMode) error { return syscall.EROFS }

func (fs *Fs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if flag != os.O_RDONLY {
		return nil, &os.PathError{Op: "open", Path: name, Err: syscall.EPERM}
	}

	return fs.Open(name)
}

func (fs *Fs) Remove(name string) error { return syscall.EROFS }

func (fs *Fs) RemoveAll(path string) error { return syscall.EROFS }

func (fs *Fs) Rename(oldname string, newname string) error { return syscall.EROFS }

func (fs *Fs) Stat(name string) (os.FileInfo, error) {
	f, ok := fs.files[name]
	if !ok {
		return nil, &os.PathError{Op: "stat", Path: name, Err: syscall.ENOENT}
	}

	return f.h.FileInfo(), nil
}

func (fs *Fs) Chmod(name string, mode os.FileMode) error { return syscall.EROFS }

func (fs *Fs) Chtimes(name string, atime time.Time, mtime time.Time) error { return syscall.EROFS }
