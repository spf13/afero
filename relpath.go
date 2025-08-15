// Copyright (C) 2024+ James Shubin
// Written by James Shubin <james@shubin.ca>
// This filesystem implementation is based on the afero.BasePathFs code.
package afero

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	_ Lstater        = (*RelPathFs)(nil)
	_ fs.ReadDirFile = (*RelPathFile)(nil)
	_ fs.ReadDirFile = readDirFile{}
)

// RelPathFs removes a prefix from all operations to a given path within an Fs.
// The given file name to the operations on this Fs will have a prefix removed
// before calling the base Fs.
//
// When initializing it with "/", a call to `/foo` turns into `foo`.
//
// Note that it does not clean the error messages on return, so you may reveal
// the real path on errors.
type RelPathFs struct {
	source Fs

	prefix string
}

type RelPathFile struct {
	File

	prefix string
}

func (obj *RelPathFile) Name() string {
	sourcename := obj.File.Name()
	//return strings.TrimPrefix(sourcename, filepath.Clean(obj.prefix))
	return filepath.Clean(obj.prefix) + sourcename // add prefix back on
}

func (obj *RelPathFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if rdf, ok := obj.File.(fs.ReadDirFile); ok {
		return rdf.ReadDir(n)
	}
	return readDirFile{obj.File}.ReadDir(n)
}

// NewRelPathFs creates a new RelPathFs.
func NewRelPathFs(source Fs, prefix string) Fs {
	return &RelPathFs{source: source, prefix: prefix}
}

// on a file outside the base path it returns the given file name and an error,
// else the given file with the base path removed
func (obj *RelPathFs) RealPath(name string) (string, error) {
	if name == "/" {
		return ".", nil // special trim
	}
	if name == "" {
		return filepath.Clean(name), nil // returns a single period
	}
	path := filepath.Clean(name)         // actual path
	prefix := filepath.Clean(obj.prefix) // is often a / and we trim it off

	//if strings.HasPrefix(path, prefix) { // redundant
	path = strings.TrimPrefix(path, prefix)
	//}

	return path, nil
}

func (obj *RelPathFs) Chtimes(name string, atime, mtime time.Time) (err error) {
	if name, err = obj.RealPath(name); err != nil {
		return &os.PathError{Op: "chtimes", Path: name, Err: err}
	}
	return obj.source.Chtimes(name, atime, mtime)
}

func (obj *RelPathFs) Chmod(name string, mode os.FileMode) (err error) {
	if name, err = obj.RealPath(name); err != nil {
		return &os.PathError{Op: "chmod", Path: name, Err: err}
	}
	return obj.source.Chmod(name, mode)
}

func (obj *RelPathFs) Chown(name string, uid, gid int) (err error) {
	if name, err = obj.RealPath(name); err != nil {
		return &os.PathError{Op: "chown", Path: name, Err: err}
	}
	return obj.source.Chown(name, uid, gid)
}

func (obj *RelPathFs) Name() string {
	return "RelPathFs"
}

func (obj *RelPathFs) Stat(name string) (fi os.FileInfo, err error) {
	if name, err = obj.RealPath(name); err != nil {
		return nil, &os.PathError{Op: "stat", Path: name, Err: err}
	}
	return obj.source.Stat(name)
}

func (obj *RelPathFs) Rename(oldname, newname string) (err error) {
	if oldname, err = obj.RealPath(oldname); err != nil {
		return &os.PathError{Op: "rename", Path: oldname, Err: err}
	}
	if newname, err = obj.RealPath(newname); err != nil {
		return &os.PathError{Op: "rename", Path: newname, Err: err}
	}
	return obj.source.Rename(oldname, newname)
}

func (obj *RelPathFs) RemoveAll(name string) (err error) {
	if name, err = obj.RealPath(name); err != nil {
		return &os.PathError{Op: "remove_all", Path: name, Err: err}
	}
	return obj.source.RemoveAll(name)
}

func (obj *RelPathFs) Remove(name string) (err error) {
	if name, err = obj.RealPath(name); err != nil {
		return &os.PathError{Op: "remove", Path: name, Err: err}
	}
	return obj.source.Remove(name)
}

func (obj *RelPathFs) OpenFile(name string, flag int, mode os.FileMode) (f File, err error) {
	if name, err = obj.RealPath(name); err != nil {
		return nil, &os.PathError{Op: "openfile", Path: name, Err: err}
	}
	sourcef, err := obj.source.OpenFile(name, flag, mode)
	if err != nil {
		return nil, err
	}
	return &RelPathFile{File: sourcef, prefix: obj.prefix}, nil
}

func (obj *RelPathFs) Open(name string) (f File, err error) {
	if name, err = obj.RealPath(name); err != nil {
		return nil, &os.PathError{Op: "open", Path: name, Err: err}
	}
	sourcef, err := obj.source.Open(name)
	if err != nil {
		return nil, err
	}
	return &RelPathFile{File: sourcef, prefix: obj.prefix}, nil
}

func (obj *RelPathFs) Mkdir(name string, mode os.FileMode) (err error) {
	if name, err = obj.RealPath(name); err != nil {
		return &os.PathError{Op: "mkdir", Path: name, Err: err}
	}
	return obj.source.Mkdir(name, mode)
}

func (obj *RelPathFs) MkdirAll(name string, mode os.FileMode) (err error) {
	if name, err = obj.RealPath(name); err != nil {
		return &os.PathError{Op: "mkdir", Path: name, Err: err}
	}
	return obj.source.MkdirAll(name, mode)
}

func (obj *RelPathFs) Create(name string) (f File, err error) {
	if name, err = obj.RealPath(name); err != nil {
		return nil, &os.PathError{Op: "create", Path: name, Err: err}
	}
	sourcef, err := obj.source.Create(name)
	if err != nil {
		return nil, err
	}
	return &RelPathFile{File: sourcef, prefix: obj.prefix}, nil
}

func (obj *RelPathFs) LstatIfPossible(name string) (os.FileInfo, bool, error) {
	name, err := obj.RealPath(name)
	if err != nil {
		return nil, false, &os.PathError{Op: "lstat", Path: name, Err: err}
	}
	if lstater, ok := obj.source.(Lstater); ok {
		return lstater.LstatIfPossible(name)
	}
	fi, err := obj.source.Stat(name)
	return fi, false, err
}

func (obj *RelPathFs) SymlinkIfPossible(oldname, newname string) error {
	oldname, err := obj.RealPath(oldname)
	if err != nil {
		return &os.LinkError{Op: "symlink", Old: oldname, New: newname, Err: err}
	}
	newname, err = obj.RealPath(newname)
	if err != nil {
		return &os.LinkError{Op: "symlink", Old: oldname, New: newname, Err: err}
	}
	if linker, ok := obj.source.(Linker); ok {
		return linker.SymlinkIfPossible(oldname, newname)
	}
	return &os.LinkError{Op: "symlink", Old: oldname, New: newname, Err: ErrNoSymlink}
}

func (obj *RelPathFs) ReadlinkIfPossible(name string) (string, error) {
	name, err := obj.RealPath(name)
	if err != nil {
		return "", &os.PathError{Op: "readlink", Path: name, Err: err}
	}
	if reader, ok := obj.source.(LinkReader); ok {
		return reader.ReadlinkIfPossible(name)
	}
	return "", &os.PathError{Op: "readlink", Path: name, Err: ErrNoReadlink}
}
