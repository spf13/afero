package afero

import (
	"io/fs"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/diskfs/go-diskfs/filesystem"
)

type DiskFsFile struct {
	filesystem.File

	name string
}

var _ File = (*DiskFsFile)(nil)

func (f *DiskFsFile) Name() string {
	return f.name
}

func (f *DiskFsFile) ReadAt([]byte, int64) (int, error) {
	return -1, syscall.EINVAL
}

func (f *DiskFsFile) Readdir(int) ([]fs.FileInfo, error) {
	return nil, syscall.EINVAL
}

func (f *DiskFsFile) Readdirnames(int) ([]string, error) {
	return nil, syscall.EINVAL
}

func (f *DiskFsFile) Stat() (fs.FileInfo, error) {
	return nil, syscall.EINVAL
}

func (f *DiskFsFile) Sync() error {
	return nil
}

func (f *DiskFsFile) Truncate(int64) error {
	return syscall.EINVAL
}

func (f *DiskFsFile) WriteAt(p []byte, off int64) (int, error) {
	if _, err := f.File.Seek(off, 0); err != nil {
		return -1, err
	}
	return f.File.Write(p)
}

func (f *DiskFsFile) WriteString(s string) (int, error) {
	return f.File.Write([]byte(s))
}

type DiskFs struct {
	source filesystem.FileSystem
}

func NewDiskFs(source filesystem.FileSystem) Fs {
	return &DiskFs{
		source: source,
	}
}

func (fs *DiskFs) Name() string { return "DiskFs" }

func (fs *DiskFs) Create(name string) (File, error) {
	f, e := fs.source.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC)
	if f == nil {
		// while this looks strange, we need to return a bare nil (of type nil) not
		// a nil value of type *os.File or nil won't be nil
		return nil, e
	}
	return &DiskFsFile{File: f, name: path.Base(name)}, e
}

func (fs *DiskFs) Mkdir(name string, _ os.FileMode) error {
	return fs.source.Mkdir(name)
}

func (fs *DiskFs) MkdirAll(path string, _ os.FileMode) error {
	return syscall.EINVAL
}

func (fs *DiskFs) Open(name string) (File, error) {
	f, e := fs.source.OpenFile(name, os.O_RDONLY)
	if f == nil {
		// while this looks strange, we need to return a bare nil (of type nil) not
		// a nil value of type *os.File or nil won't be nil
		return nil, e
	}
	return &DiskFsFile{File: f, name: path.Base(name)}, e
}

func (fs *DiskFs) OpenFile(name string, flag int, _ os.FileMode) (File, error) {
	f, e := fs.source.OpenFile(name, flag)
	if f == nil {
		// while this looks strange, we need to return a bare nil (of type nil) not
		// a nil value of type *os.File or nil won't be nil
		return nil, e
	}
	return &DiskFsFile{File: f, name: path.Base(name)}, e
}

func (fs *DiskFs) Remove(_ string) error {
	return syscall.EPERM
}

func (fs *DiskFs) RemoveAll(_ string) error {
	return syscall.EPERM
}

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

func (fs *DiskFs) Chmod(_ string, _ os.FileMode) error {
	return syscall.EPERM
}

func (fs *DiskFs) Chown(_ string, _, _ int) error {
	return syscall.EPERM
}

func (fs *DiskFs) Chtimes(_ string, _, _ time.Time) error {
	return syscall.EPERM
}
