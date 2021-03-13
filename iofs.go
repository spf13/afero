// +build go1.16

package afero

import (
	"io/fs"
	"path"
)

// IOFS adopts afero.Fs to stdlib io/fs.FS
type IOFS struct {
	Fs
}

func NewIOFS(fs Fs) IOFS {
	return IOFS{Fs: fs}
}

var (
	_ fs.FS         = IOFS{}
	_ fs.GlobFS     = IOFS{}
	_ fs.ReadDirFS  = IOFS{}
	_ fs.ReadFileFS = IOFS{}
	_ fs.StatFS     = IOFS{}
	_ fs.SubFS      = IOFS{}
)

func (iofs IOFS) Open(name string) (fs.File, error) {
	const op = "open"

	// by convention for fs.FS implementations we should perform this check
	if !fs.ValidPath(name) {
		return nil, iofs.wrapError(op, name, fs.ErrInvalid)
	}

	file, err := iofs.Fs.Open(name)
	if err != nil {
		return nil, iofs.wrapError(op, name, err)
	}

	// file should implement fs.ReadDirFile
	if _, ok := file.(fs.ReadDirFile); !ok {
		file = readDirFile{file}
	}

	return file, nil
}

func (iofs IOFS) Glob(pattern string) ([]string, error) {
	const op = "glob"

	// afero.Glob does not perform this check but it's required for implementations
	if _, err := path.Match(pattern, ""); err != nil {
		return nil, iofs.wrapError(op, pattern, err)
	}

	items, err := Glob(iofs.Fs, pattern)
	if err != nil {
		return nil, iofs.wrapError(op, pattern, err)
	}

	return items, nil
}

func (iofs IOFS) ReadDir(name string) ([]fs.DirEntry, error) {
	items, err := ReadDir(iofs.Fs, name)
	if err != nil {
		return nil, iofs.wrapError("readdir", name, err)
	}

	ret := make([]fs.DirEntry, len(items))
	for i := range items {
		ret[i] = dirEntry{items[i]}
	}

	return ret, nil
}

func (iofs IOFS) ReadFile(name string) ([]byte, error) {
	const op = "readfile"

	if !fs.ValidPath(name) {
		return nil, iofs.wrapError(op, name, fs.ErrInvalid)
	}

	bytes, err := ReadFile(iofs.Fs, name)
	if err != nil {
		return nil, iofs.wrapError(op, name, err)
	}

	return bytes, nil
}

func (iofs IOFS) Sub(dir string) (fs.FS, error) { return IOFS{NewBasePathFs(iofs.Fs, dir)}, nil }

func (IOFS) wrapError(op, path string, err error) error {
	if _, ok := err.(*fs.PathError); ok {
		return err // don't need to wrap again
	}

	return &fs.PathError{
		Op:   op,
		Path: path,
		Err:  err,
	}
}

// dirEntry provides adapter from os.FileInfo to fs.DirEntry
type dirEntry struct {
	fs.FileInfo
}

var _ fs.DirEntry = dirEntry{}

func (d dirEntry) Type() fs.FileMode { return d.FileInfo.Mode().Type() }

func (d dirEntry) Info() (fs.FileInfo, error) { return d.FileInfo, nil }

// readDirFile provides adapter from afero.File to fs.ReadDirFile needed for correct Open
type readDirFile struct {
	File
}

var _ fs.ReadDirFile = readDirFile{}

func (r readDirFile) ReadDir(n int) ([]fs.DirEntry, error) {
	items, err := r.File.Readdir(n)
	if err != nil {
		return nil, err
	}

	ret := make([]fs.DirEntry, len(items))
	for i := range items {
		ret[i] = dirEntry{items[i]}
	}

	return ret, nil
}
