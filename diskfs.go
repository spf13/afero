package afero

import (
	"io"
	"io/fs"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/diskfs/go-diskfs/filesystem"
)

// DiskFsFile is a wrapper around filesystem.File to implement the afero File interface.
type DiskFsFile struct {
	filesystem.File

	name string
}

var _ File = (*DiskFsFile)(nil)

// Name returns the name of the file.
func (f *DiskFsFile) Name() string {
	return f.name
}

func (f *DiskFsFile) ReadAt(p []byte, off int64) (int, error) {
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return -1, err
	}

	return f.Read(p)
}

// Readdir is not supported and returns an error.
func (f *DiskFsFile) Readdir(int) ([]fs.FileInfo, error) {
	return nil, syscall.EINVAL
}

// Readdirnames is not supported and returns an error.
func (f *DiskFsFile) Readdirnames(int) ([]string, error) {
	return nil, syscall.EINVAL
}

// Stat is not supported and returns an error.
func (f *DiskFsFile) Stat() (fs.FileInfo, error) {
	return nil, syscall.EINVAL
}

// Sync is a no-op for DiskFsFile.
func (f *DiskFsFile) Sync() error {
	return nil
}

// Truncate truncates the file to a specified length using naive approach.
func (f *DiskFsFile) Truncate(off int64) error {
	if _, err := f.WriteAt([]byte{0}, off); err != nil {
		return err
	}
	return nil
}

func (f *DiskFsFile) WriteAt(p []byte, off int64) (int, error) {
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return -1, err
	}
	return f.Write(p)
}

func (f *DiskFsFile) WriteString(s string) (int, error) {
	return f.Write([]byte(s))
}

// DiskFs is an implementation of afero Fs interface using diskfs.
type DiskFs struct {
	source filesystem.FileSystem
}

// NewDiskFs creates a new DiskFs with the provided source filesystem.
func NewDiskFs(source filesystem.FileSystem) Fs {
	return &DiskFs{
		source: source,
	}
}

func (fs *DiskFs) Name() string {
	return "DiskFs"
}

func (fs *DiskFs) Create(name string) (File, error) {
	f, err := fs.source.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC)
	if f == nil {
		return nil, err
	}
	return &DiskFsFile{File: f, name: path.Base(name)}, err
}

// Mkdir and MkdirAll behave identically, creating a directory and all its parent directories.
func (fs *DiskFs) Mkdir(name string, _ os.FileMode) error {
	return fs.source.Mkdir(name)
}

// MkdirAll and Mkdir behave identically, creating a directory and all its parent directories.
func (fs *DiskFs) MkdirAll(path string, _ os.FileMode) error {
	return fs.source.Mkdir(path)
}

func (fs *DiskFs) Open(name string) (File, error) {
	f, err := fs.source.OpenFile(name, os.O_RDONLY)
	if f == nil {
		return nil, err
	}
	return &DiskFsFile{File: f, name: path.Base(name)}, err
}

func (fs *DiskFs) OpenFile(name string, flag int, _ os.FileMode) (File, error) {
	f, err := fs.source.OpenFile(name, flag)
	if f == nil {
		return nil, err
	}
	return &DiskFsFile{File: f, name: path.Base(name)}, err
}

// Remove is not supported and returns an error.
func (fs *DiskFs) Remove(_ string) error {
	return syscall.EPERM
}

// RemoveAll is not supported and returns an error.
func (fs *DiskFs) RemoveAll(_ string) error {
	return syscall.EPERM
}

// Rename is not supported and returns an error.
func (fs *DiskFs) Rename(_, _ string) error {
	return syscall.EPERM
}

func (fs *DiskFs) Stat(name string) (os.FileInfo, error) {
	infos, err := fs.source.ReadDir(path.Dir(name))
	if err != nil {
		return nil, err
	}

	for _, info := range infos {
		if info.Name() == path.Base(name) {
			return info, nil
		}
	}

	return nil, syscall.EINVAL
}

// Chmod is not supported and returns an error.
func (fs *DiskFs) Chmod(_ string, _ os.FileMode) error {
	return syscall.EPERM
}

// Chown is not supported and returns an error.
func (fs *DiskFs) Chown(_ string, _, _ int) error {
	return syscall.EPERM
}

// Chtimes is not supported and returns an error.
func (fs *DiskFs) Chtimes(_ string, _, _ time.Time) error {
	return syscall.EPERM
}
