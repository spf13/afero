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
	files map[string]map[string]*fsEntry
}

type fsEntry struct {
	h    *tar.Header
	data []byte
}

func splitpath(name string) (dir, file string) {
	name = filepath.ToSlash(name)
	if len(name) == 0 || name[0] != '/' {
		name = "/" + name
	}
	name = filepath.Clean(name)
	dir, file = filepath.Split(name)
	dir = filepath.Clean(dir)
	return
}

func New(t *tar.Reader) *Fs {
	fs := &Fs{files: make(map[string]map[string]*fsEntry)}
	for {
		hdr, err := t.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil
		}

		d, f := splitpath(hdr.Name)
		if _, ok := fs.files[d]; !ok {
			fs.files[d] = make(map[string]*fsEntry)
		}

		var buf bytes.Buffer
		size, err := buf.ReadFrom(t)
		if err != nil {
			panic("tarfs: reading from tar:" + err.Error())
		}

		if size != hdr.Size {
			panic("tarfs: size mismatch")
		}

		file := &fsEntry{
			h:    hdr,
			data: buf.Bytes(),
		}
		fs.files[d][f] = file

	}

	if fs.files[afero.FilePathSeparator] == nil {
		fs.files[afero.FilePathSeparator] = make(map[string]*fsEntry)
	}
	// Add a pseudoroot
	fs.files[afero.FilePathSeparator][""] = &fsEntry{
		h: &tar.Header{
			Name:     afero.FilePathSeparator,
			Typeflag: tar.TypeDir,
			Size:     0,
		},
		data: nil,
	}

	return fs
}

func (fs *Fs) Open(name string) (afero.File, error) {
	d, f := splitpath(name)
	if _, ok := fs.files[d]; !ok {
		return nil, &os.PathError{Op: "open", Path: name, Err: syscall.ENOENT}
	}

	file, ok := fs.files[d][f]
	if !ok {
		return nil, &os.PathError{Op: "open", Path: name, Err: syscall.ENOENT}
	}

	nf := File{
		h:    file.h,
		data: bytes.NewReader(file.data),
		fs:   fs,
	}

	return &nf, nil
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
	d, f := splitpath(name)
	if _, ok := fs.files[d]; !ok {
		return nil, &os.PathError{Op: "stat", Path: name, Err: syscall.ENOENT}
	}

	file, ok := fs.files[d][f]
	if !ok {
		return nil, &os.PathError{Op: "stat", Path: name, Err: syscall.ENOENT}
	}

	return file.h.FileInfo(), nil
}

func (fs *Fs) Chmod(name string, mode os.FileMode) error { return syscall.EROFS }

func (fs *Fs) Chown(name string, uid, gid int) error { return syscall.EROFS }

func (fs *Fs) Chtimes(name string, atime time.Time, mtime time.Time) error { return syscall.EROFS }
