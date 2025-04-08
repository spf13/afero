package ossfs

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/spf13/afero"
	"github.com/spf13/afero/ossfs/internal/utils"
)

const (
	defaultFileMode = 0o755
	defaultFileFlag = os.O_RDWR
)

type Fs struct {
	manager     utils.ObjectManager
	bucketName  string
	separator   string
	autoSync    bool
	openedFiles map[string]afero.File
	preloadFs   afero.Fs
	ctx         context.Context
}

func NewOssFs(accessKeyId, accessKeySecret, region, bucket string) *Fs {
	ossCfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyId, accessKeySecret)).
		WithRegion(region)

	return &Fs{
		manager: &utils.OssObjectManager{
			Client: oss.NewClient(ossCfg),
		},
		bucketName:  bucket,
		separator:   "/",
		autoSync:    true,
		openedFiles: make(map[string]afero.File),
		preloadFs:   afero.NewMemMapFs(),
		ctx:         context.Background(),
	}
}

func (fs *Fs) WithContext(ctx context.Context) *Fs {
	fs.ctx = ctx
	return fs
}

// Create creates a new empty file and open it, return the open file and error
// if any happens.
func (fs *Fs) Create(name string) (afero.File, error) {
	n := fs.normFileName(name)
	r := strings.NewReader("")
	if _, err := fs.manager.PutObject(fs.ctx, fs.bucketName, n, r); err != nil {
		return nil, err
	}
	return NewOssFile(n, defaultFileFlag, fs)
}

// Mkdir creates a directory in the filesystem, return an error if any
// happens.
func (fs *Fs) Mkdir(name string, perm os.FileMode) error {
	return fs.MkdirAll(fs.ensureAsDir(name), perm)
}

// MkdirAll creates a directory path and all parents that does not exist
// yet.
func (fs *Fs) MkdirAll(path string, perm os.FileMode) error {
	dirName := fs.ensureAsDir(path)
	r := strings.NewReader("")
	_, err := fs.manager.PutObject(fs.ctx, fs.bucketName, dirName, r)
	return err
}

// Open opens a file, returning it or an error, if any happens.
func (fs *Fs) Open(name string) (afero.File, error) {
	return fs.OpenFile(name, defaultFileFlag, defaultFileMode)
}

// OpenFile opens a file using the given flags and the given mode.
func (fs *Fs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	name = fs.normFileName(name)
	file, found := fs.openedFiles[name]
	if found && file.(*File).openFlag == flag {
		return file, nil
	}

	f, err := NewOssFile(name, flag, fs)
	if err != nil {
		return nil, err
	}

	existed := false
	existed, err = fs.manager.IsObjectExist(fs.ctx, fs.bucketName, name)
	if err != nil {
		return nil, err
	}

	if !existed && f.openFlag&os.O_CREATE == 0 {
		return nil, afero.ErrFileNotFound
	}

	if !existed && f.openFlag*os.O_CREATE != 0 {
		if _, err := fs.Create(f.name); err != nil {
			return nil, err
		}
	}

	if f.openFlag&os.O_TRUNC != 0 {
		_, err := f.fs.manager.PutObject(fs.ctx, fs.bucketName, f.name, strings.NewReader(""))
		if err != nil {
			return nil, err
		}
	}

	fs.openedFiles[name] = f

	return f, nil
}

// Remove removes a file identified by name, returning an error, if any
// happens.
func (fs *Fs) Remove(name string) error {
	return fs.manager.DeleteObject(fs.ctx, fs.bucketName, fs.normFileName(name))
}

// RemoveAll removes a directory path and any children it contains. It
// does not fail if the path does not exist (return nil).
func (fs *Fs) RemoveAll(path string) error {
	dir := fs.ensureAsDir(path)
	fis, err := fs.manager.ListAllObjects(fs.ctx, fs.bucketName, dir)
	if err != nil {
		return err
	}
	for _, fi := range fis {
		err = fs.manager.DeleteObject(fs.ctx, fs.bucketName, fi.Name())
		if err != nil {
			return err
		}
	}
	return nil
}

// Rename renames a file.
func (fs *Fs) Rename(oldname, newname string) error {
	err := fs.manager.CopyObject(fs.ctx, fs.bucketName, oldname, newname)
	if err != nil {
		return err
	}
	err = fs.manager.DeleteObject(fs.ctx, fs.bucketName, oldname)
	return err
}

// Stat returns a FileInfo describing the named file, or an error, if any
// happens.
func (fs *Fs) Stat(name string) (os.FileInfo, error) {
	fi, err := fs.manager.GetObjectMeta(fs.ctx, fs.bucketName, fs.normFileName(name))
	if err != nil {
		return nil, err
	}

	return fi, err
}

// The name of this FileSystem
func (fs *Fs) Name() string {
	return "OssFs"
}

// Chmod changes the mode of the named file to mode.
func (fs *Fs) Chmod(name string, mode os.FileMode) error {
	return errors.New("OSS: method Chmod is not implemented")
}

// Chown changes the uid and gid of the named file.
func (fs *Fs) Chown(name string, uid, gid int) error {
	return errors.New("OSS: method Chown is not implemented")
}

// Chtimes changes the access and modification times of the named file
func (fs *Fs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return errors.New("OSS: method Chtimes is not implemented")
}
