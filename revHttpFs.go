package afero

import (
	"io"
	"net/http"
	"os"
	"syscall"
	"time"
)

// ReverseHttpFile takes an http.File and matches it to File
type ReverseHttpFile struct {
	http.File
}

// Name returns the file's name
func (a ReverseHttpFile) Name() string {
	s, err := a.File.Stat()
	if err != nil {
		return "" // No errors allowed
	}
	return s.Name()
}

// WriteAt always returns a permissions error, since an http.FileSystem is read-only
func (a ReverseHttpFile) WriteAt(p []byte, off int64) (int, error) {
	return 0, syscall.EPERM
}

// ReadAt seeks, then reads
func (a ReverseHttpFile) ReadAt(p []byte, off int64) (int, error) {
	if _, err := a.File.Seek(off, io.SeekStart); err != nil {
		return 0, err
	}

	return a.File.Read(p)
}

// Readdirnames returns a list of names from Readdir
func (a ReverseHttpFile) Readdirnames(n int) ([]string, error) {
	dirs, err := a.File.Readdir(n)
	if err != nil {
		return nil, err
	}

	out := make([]string, len(dirs))
	for d := range dirs {
		out[d] = dirs[d].Name()
	}
	return out, nil
}

// Sync does nothing here
func (a ReverseHttpFile) Sync() error {
	return nil
}

// Truncate always returns a permissions error, since an http.FileSystem is read-only
func (a ReverseHttpFile) Truncate(size int64) error {
	return syscall.EPERM
}

// WriteString always returns a permissions error, since an http.FileSystem is read-only
func (a ReverseHttpFile) WriteString(s string) (int, error) {
	return 0, syscall.EPERM
}

// Write always returns a permissions error, since an http.FileSystem is read-only
func (a ReverseHttpFile) Write(n []byte) (int, error) {
	return 0, syscall.EPERM
}

// ReverseHttpFs converts an http.Filesystem into an afero Fs
type ReverseHttpFs struct {
	http.FileSystem
}

// Given an http.FileSystem, returns an Fs
func NewReverseHttpFs(fs http.FileSystem) ReverseHttpFs {
	return ReverseHttpFs{fs}
}

// Mkdir always returns a permissions error, since an http.FileSystem is read-only
func (fs ReverseHttpFs) Mkdir(n string, p os.FileMode) error {
	return syscall.EPERM
}

// MkdirAll always returns a permissions error, since an http.FileSystem is read-only
func (r ReverseHttpFs) MkdirAll(n string, p os.FileMode) error {
	return syscall.EPERM
}

// Create always returns a permissions error, since an http.FileSystem is read-only
func (fs ReverseHttpFs) Create(n string) (File, error) {
	return nil, syscall.EPERM
}

// ReadDir reads the given file as a directory
func (fs ReverseHttpFs) ReadDir(name string) ([]os.FileInfo, error) {
	f, err := fs.FileSystem.Open(name)
	if err != nil {
		return nil, err
	}
	return f.Readdir(0)
}

// Chtimes always returns a permissions error, since an http.FileSystem is read-only
func (fs ReverseHttpFs) Chtimes(n string, a, m time.Time) error {
	return syscall.EPERM
}

// Chmod always returns a permissions error, since an http.FileSystem is read-only
func (fs ReverseHttpFs) Chmod(n string, m os.FileMode) error {
	return syscall.EPERM
}

// Name returns "ReverseHttpFs"
func (fs ReverseHttpFs) Name() string {
	return "ReverseHttpFs"
}

// Stat runs Stat on the file
func (fs ReverseHttpFs) Stat(name string) (os.FileInfo, error) {
	f, err := fs.FileSystem.Open(name)
	if err != nil {
		return nil, err
	}
	return f.Stat()
}

// LstatIfPossible always fails here
func (fs ReverseHttpFs) LstatIfPossible(name string) (os.FileInfo, bool, error) {
	fi, err := fs.Stat(name)
	return fi, false, err
}

// Rename always returns a permissions error, since an http.FileSystem is read-only
func (fs ReverseHttpFs) Rename(o, n string) error {
	return syscall.EPERM
}

// RemoveAll always returns a permissions error, since an http.FileSystem is read-only
func (fs ReverseHttpFs) RemoveAll(p string) error {
	return syscall.EPERM
}

// Remove always returns a permissions error, since an http.FileSystem is read-only
func (fs ReverseHttpFs) Remove(n string) error {
	return syscall.EPERM
}

// OpenFile opens the file (readonly)
func (fs ReverseHttpFs) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	if flag&(os.O_WRONLY|syscall.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC) != 0 {
		return nil, syscall.EPERM
	}

	return fs.Open(name)
}

// Open opens the given file (readonly)
func (fs ReverseHttpFs) Open(n string) (File, error) {
	f, err := fs.FileSystem.Open(n)
	return ReverseHttpFile{f}, err
}
