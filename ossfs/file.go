package ossfs

import (
	"fmt"
	"io"
	"os"
	"slices"
	"syscall"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/spf13/afero"
)

type File struct {
	name        string
	fs          *Fs
	openFlag    int
	offset      int64
	fi          *FileInfo
	dirty       bool
	closed      bool
	isDir       bool
	preloaded   bool
	preloadedFd afero.File
}

func NewOssFile(name string, flag int, fs *Fs) (*File, error) {
	return &File{
		name:        name,
		fs:          fs,
		openFlag:    flag,
		offset:      0,
		dirty:       false,
		closed:      false,
		isDir:       fs.isDir(name),
		preloaded:   false,
		preloadedFd: nil,
	}, nil
}

func (f *File) preload() error {
	preloadedFs, err := f.fs.preloadFs.Create(f.name)
	if err != nil {
		return err
	}
	f.preloadedFd = preloadedFs
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

func (f *File) getFileInfo() (*FileInfo, error) {
	if f.dirty {
		if err := f.freshFileInfo(); err != nil {
			return nil, err
		}
	}
	return f.fi, nil
}

func (f *File) hasFlag(flag int) bool {
	return f.openFlag&flag != 0
}

func (f *File) isReadable() bool {
	return !f.closed && (f.hasFlag(os.O_RDONLY) || f.hasFlag(os.O_RDWR))
}

func (f *File) isWriteable() bool {
	return !f.closed && (f.hasFlag(os.O_WRONLY) || f.hasFlag(os.O_RDWR))
}

func (f *File) isAppendOnly() bool {
	return f.isWriteable() && f.hasFlag(os.O_APPEND)
}

func (f *File) shouldTruncate() bool {
	return f.hasFlag(os.O_TRUNC)
}

func (f *File) shouldCreateIfNotExists() bool {
	return f.hasFlag(os.O_CREATE)
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
	req := &oss.GetObjectRequest{
		Bucket:        oss.Ptr(f.fs.bucketName),
		Key:           oss.Ptr(f.name),
		Range:         oss.Ptr(fmt.Sprintf("bytes=%v-%v", off, off+int64(len(p)))),
		RangeBehavior: oss.Ptr("standard"),
	}
	res, err := f.fs.client.GetObject(f.fs.ctx, req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	buf, _ := io.ReadAll(res.Body)
	p = slices.Concat(p, buf)
	_ = p
	return len(buf), nil
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
	}
	if newOffset < 0 || newOffset > max {
		return 0, syscall.ERANGE
	}
	f.offset = newOffset
	return f.offset, nil
}

func (f *File) Append(p []byte) (int, error) {
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
		return f.Append(p)
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

func (f *File) Name() string {
	return f.name
}

func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	if !f.isReadable() {
		return nil, syscall.EPERM
	}

	req := &oss.ListObjectsV2Request{
		Bucket:  oss.Ptr(f.fs.bucketName),
		Prefix:  oss.Ptr(f.fs.ensureAsDir(f.name)),
		MaxKeys: int32(count),
	}

	p := f.fs.client.NewListObjectsV2Paginator(req)

	var fis []os.FileInfo

	for p.HasNext() {
		page, err := p.NextPage(f.fs.ctx)
		if err != nil {
			return nil, err
		}
		for _, objProps := range page.Contents {
			fi := NewFileInfoWithObjProp(oss.ToString(objProps.Key), f.fs, objProps)
			fis = append(fis, fi)
		}
	}

	return fis, nil
}

func (f *File) Readdirnames(n int) ([]string, error) {
	if !f.isReadable() {
		return nil, syscall.EPERM
	}

	fis, err := f.Readdir(n)
	if err != nil {
		return nil, err
	}
	var dirNames []string
	for _, fi := range fis {
		dirNames = append(dirNames, fi.Name())
	}

	return dirNames, nil
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
		if _, err := f.fs.putObjectReader(f.name, f); err != nil {
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
