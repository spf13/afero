package ossfs

import (
	"context"
	"errors"
	"os"
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
	utils.ObjectManager
	bucketName  string
	separator   string
	autoSync    bool
	openedFiles map[string]afero.File
	preloadFs   afero.Fs
}

func NewOssFs(accessKeyId, accessKeySecret, region, bucket string) *Fs {
	ossCfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyId, accessKeySecret)).
		WithRegion(region)

	return &Fs{
		ObjectManager: &utils.OssObjectManager{
			Client: oss.NewClient(ossCfg),
			Ctx:    context.TODO(),
		},
		bucketName:  bucket,
		separator:   "/",
		autoSync:    true,
		openedFiles: make(map[string]afero.File),
		preloadFs:   afero.NewMemMapFs(),
	}
}

// Create creates a new empty file and open it, return the open file and error
// if any happens.
func (fs *Fs) Create(name string) (afero.File, error) {
	if _, err := fs.putObjectStr(name, ""); err != nil {
		return nil, err
	}
	return fs.OpenFile(name, defaultFileFlag, defaultFileMode)
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
	_, err := fs.putObjectStr(dirName, "")
	return err
}

// Open opens a file, returning it or an error, if any happens.
func (fs *Fs) Open(name string) (afero.File, error) {
	return fs.OpenFile(name, defaultFileFlag, defaultFileMode)
}

// OpenFile opens a file using the given flags and the given mode.
func (fs *Fs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	file, found := fs.openedFiles[name]
	if found && file.(*File).openFlag == flag {
		return file, nil
	}

	f, err := NewOssFile(name, flag, fs)
	if err != nil {
		return nil, err
	}

	existed := false
	existed, err = fs.isObjectExists(name)
	if err != nil {
		return nil, err
	}

	if !existed && f.shouldCreateIfNotExists() {
		if _, err := fs.Create(f.name); err != nil {
			return nil, err
		}
	}

	if f.shouldTruncate() {
		if _, err := f.fs.putObjectStr(f.name, ""); err != nil {
			return nil, err
		}
	}

	fs.openedFiles[name] = f

	return f, nil
}

// Remove removes a file identified by name, returning an error, if any
// happens.
func (fs *Fs) Remove(name string) error {
	req := &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(fs.bucketName),
		Key:    oss.Ptr(name),
	}
	_, err := fs.client.DeleteObject(fs.ctx, req)
	return err
}

// RemoveAll removes a directory path and any children it contains. It
// does not fail if the path does not exist (return nil).
func (fs *Fs) RemoveAll(path string) error {
	req := &oss.ListObjectsV2Request{
		Bucket: oss.Ptr(fs.bucketName),
		Prefix: oss.Ptr(fs.ensureAsDir(path)),
	}
	p := fs.client.NewListObjectsV2Paginator(req)
	for p.HasNext() {
		page, err := p.NextPage(fs.ctx)
		if err != nil {
			return err
		}
		for _, obj := range page.Contents {
			fs.Remove(oss.ToString(obj.Key))
		}
	}
	return nil
}

// Rename renames a file.
func (fs *Fs) Rename(oldname, newname string) error {
	_, err := fs.copyObject(oldname, newname)
	if err != nil {
		return err
	}
	_, err = fs.deleteObject(oldname)
	return err
}

// Stat returns a FileInfo describing the named file, or an error, if any
// happens.
func (fs *Fs) Stat(name string) (*FileInfo, error) {
	req := &oss.HeadObjectRequest{
		Bucket: oss.Ptr(fs.bucketName),
		Key:    oss.Ptr(name),
	}
	res, err := fs.client.HeadObject(fs.ctx, req)
	if err != nil {
		return nil, err
	}
	return NewFileInfo(name, fs, res), nil
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
