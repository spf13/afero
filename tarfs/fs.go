// package tarfs implements a read-only in-memory representation of a tar archive
package tarfs

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
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
			h:      hdr,
			at:     0,
			closed: true,
		}

		var buf bytes.Buffer
		io.Copy(&buf, t)

		fs.files[filepath.Clean(hdr.Name)] = f

	}

	return fs
}

func (fs *Fs) Open(name string) (afero.File, error) {
	panic("not implemented")
}

func (fs *Fs) Name() string { return "tarfs" }

func (fs *Fs) Create(name string) (afero.File, error) {
	panic("not implemented")
}

func (fs *Fs) Mkdir(name string, perm os.FileMode) error {
	panic("not implemented")
}

func (fs *Fs) MkdirAll(path string, perm os.FileMode) error {
	panic("not implemented")
}

func (fs *Fs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	panic("not implemented")
}

func (fs *Fs) Remove(name string) error {
	panic("not implemented")
}

func (fs *Fs) RemoveAll(path string) error {
	panic("not implemented")
}

func (fs *Fs) Rename(oldname string, newname string) error {
	panic("not implemented")
}

func (fs *Fs) Stat(name string) (os.FileInfo, error) {
	panic("not implemented")
}

func (fs *Fs) Chmod(name string, mode os.FileMode) error {
	panic("not implemented")
}

func (fs *Fs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	panic("not implemented")
}
