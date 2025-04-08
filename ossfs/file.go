package ossfs

import (
	"errors"
	"io"
	"os"
	"strconv"
	"syscall"

	"github.com/spf13/afero"
)

type File struct {
	name        string
	fs          *Fs
	openFlag    int
	offset      int64
	fi          os.FileInfo
	dirty       bool
	closed      bool
	isDir       bool
	preloaded   bool
	preloadedFd afero.File
}

func NewOssFile(name string, flag int, fs *Fs) (*File, error) {
	return &File{
		name:        fs.normFileName(name),
		fs:          fs,
		openFlag:    flag,
		offset:      0,
		dirty:       false,
		closed:      false,
		isDir:       fs.isDir(fs.normFileName(name)),
		preloaded:   false,
		preloadedFd: nil,
	}, nil
}

func (f *File) preload() error {
	pfs := f.fs.preloadFs
	if _, err := pfs.Stat(f.name); err == nil {
		if e := pfs.Remove(f.name); e != nil {
			return e
		}
	}
	pfd, err := f.fs.preloadFs.Create(f.name)
	if err != nil {
		return err
	}
	r, clean, e := f.fs.manager.GetObject(f.fs.ctx, f.fs.bucketName, f.name)
	if e != nil {
		return e
	}
	defer clean()
	p, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	_, err = pfd.WriteAt(p, 0)
	if err != nil {
		return err
	}
	pfd.Seek(f.offset, io.SeekStart)
	f.preloadedFd = pfd
	return nil
}

func (f *File) freshFileInfo() error {
	fi, err := f.fs.Stat(f.name)
	if err != nil {
		return err
	}
	f.fi = fi
	f.dirty = false
	return nil
}

func (f *File) getFileInfo() (os.FileInfo, error) {
	if f.dirty {
		if err := f.freshFileInfo(); err != nil {
			return nil, err
		}
	}
	return f.fi, nil
}

func (f *File) isReadable() bool {
	return !f.closed && (f.openFlag == os.O_RDONLY || f.openFlag == os.O_RDWR)
}

func (f *File) isWriteable() bool {
	return !f.closed && (f.openFlag == os.O_WRONLY || f.openFlag == os.O_RDWR)
}

func (f *File) isAppendOnly() bool {
	return f.isWriteable() && f.openFlag&os.O_APPEND != 0
}

func (f *File) Read(p []byte) (int, error) {
	if !f.isReadable() || f.isDir {
		return 0, syscall.EPERM
	}
	n, err := f.ReadAt(p, f.offset)
	if err != nil {
		return 0, err
	}
	f.offset += int64(n)
	return n, err
}

func (f *File) ReadAt(p []byte, off int64) (int, error) {
	if !f.isReadable() || f.isDir {
		return 0, syscall.EPERM
	}
	reader, cleanUp, err := f.fs.manager.GetObjectPart(f.fs.ctx, f.fs.bucketName, f.name, off, off+int64(len(p)))
	if err != nil {
		return 0, err
	}
	defer cleanUp()
	return reader.Read(p)
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	if (!f.isReadable() && !f.isWriteable()) || f.isDir {
		return 0, syscall.EPERM
	}
	fi, err := f.getFileInfo()
	if err != nil {
		return 0, err
	}
	max := fi.Size()
	var newOffset int64
	switch whence {
	case io.SeekCurrent:
		newOffset = f.offset + offset
	case io.SeekStart:
		newOffset = offset
	case io.SeekEnd:
		newOffset = max + offset
	default:
		return 0, errors.New("Invalid whence value: " + strconv.Itoa(whence))
	}
	if newOffset < 0 || newOffset > max {
		return 0, afero.ErrOutOfRange
	}
	f.offset = newOffset
	return f.offset, nil
}

func (f *File) doAppend(p []byte) (int, error) {
	if !f.isWriteable() {
		return 0, syscall.EPERM
	}
	fi, err := f.getFileInfo()
	if err != nil {
		return 0, err
	}
	return f.doWriteAt(p, fi.Size())
}

func (f *File) Write(p []byte) (int, error) {
	if !f.isWriteable() {
		return 0, syscall.EPERM
	}
	if f.isAppendOnly() {
		return f.doAppend(p)
	}
	n, e := f.doWriteAt(p, f.offset)
	if e != nil {
		return 0, e
	}
	f.offset += int64(n)
	return n, e
}

func (f *File) doWriteAt(p []byte, off int64) (int, error) {
	if f.isDir {
		return 0, syscall.EPERM
	}

	if !f.preloaded {
		if err := f.preload(); err != nil {
			return 0, err
		}
	}

	n, e := f.preloadedFd.WriteAt(p, off)
	f.dirty = true
	if f.fs.autoSync {
		f.Sync()
	}
	return n, e
}

func (f *File) WriteAt(p []byte, off int64) (int, error) {
	if !f.isWriteable() || f.isAppendOnly() {
		return 0, syscall.EPERM
	}
	return f.doWriteAt(p, off)
}

func (f *File) Close() error {
	f.Sync()
	f.closed = true
	delete(f.fs.openedFiles, f.name)
	if f.preloaded {
		err := f.fs.preloadFs.Remove(f.name)
		if err != nil {
			return err
		}
		err = f.preloadedFd.Close()
		if err != nil {
			return err
		}
		f.preloadedFd = nil
		f.preloaded = false
	}
	return nil
}

func (f *File) Name() string {
	return f.name
}

func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	if !f.isReadable() {
		return nil, syscall.EPERM
	}

	fis, err := f.fs.manager.ListObjects(f.fs.ctx, f.fs.bucketName, f.fs.ensureAsDir(f.name), count)
	return fis, err
}

func (f *File) Readdirnames(n int) ([]string, error) {
	if !f.isReadable() {
		return nil, syscall.EPERM
	}

	fis, err := f.Readdir(n)
	if err != nil {
		return nil, err
	}
	var fNames []string
	for _, fi := range fis {
		fNames = append(fNames, fi.Name())
	}

	return fNames, nil
}

func (f *File) Stat() (os.FileInfo, error) {
	if f.dirty {
		err := f.freshFileInfo()
		if err != nil {
			return nil, err
		}
	}
	return f.fi, nil
}

func (f *File) Sync() error {
	if f.preloaded {
		if _, err := f.fs.manager.PutObject(f.fs.ctx, f.fs.bucketName, f.name, f.preloadedFd); err != nil {
			return err
		}
	}
	if f.dirty {
		if err := f.freshFileInfo(); err != nil {
			return err
		}
	}
	return nil
}

func (f *File) Truncate(size int64) error {
	if !f.isWriteable() || f.isDir {
		return syscall.EPERM
	}
	_, err := f.WriteAt([]byte(""), 0)
	return err
}

func (f *File) WriteString(s string) (int, error) {
	return f.Write([]byte(s))
}
