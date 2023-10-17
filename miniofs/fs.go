package minio

import (
	"context"
	"errors"
	"github.com/minio/minio-go/v7"
	"log"
	"os"
	"strings"
	"time"
)

const (
	defaultFileMode = 0o755
	gsPrefix        = "gs://"
)

// Fs is a Fs implementation that uses functions provided by google cloud storage
type Fs struct {
	ctx       context.Context
	client    *minio.Client
	separator string

	buckets         map[string]*minio.BucketInfo
	rawMinioObjects map[string]*MinioFile

	autoRemoveEmptyFolders bool // trigger for creating "virtual folders" (not required by Minio)
}

func NewMinioFs(ctx context.Context, client *minio.Client) *Fs {
	return NewMinioFsWithSeparator(ctx, client, "/")
}

func NewMinioFsWithSeparator(ctx context.Context, client *minio.Client, folderSep string) *Fs {
	return &Fs{
		ctx:                    ctx,
		client:                 client,
		separator:              folderSep,
		buckets:                make(map[string]*minio.BucketInfo),
		rawMinioObjects:        make(map[string]*MinioFile),
		autoRemoveEmptyFolders: true,
	}
}

// normSeparators will normalize all "\\" and "/" to the provided separator
func (fs *Fs) normSeparators(s string) string {
	return strings.Replace(strings.Replace(s, "\\", fs.separator, -1), "/", fs.separator, -1)
}

func (fs *Fs) ensureTrailingSeparator(s string) string {
	if len(s) > 0 && !strings.HasSuffix(s, fs.separator) {
		return s + fs.separator
	}
	return s
}

func (fs *Fs) ensureNoLeadingSeparator(s string) string {
	if len(s) > 0 && strings.HasPrefix(s, fs.separator) {
		s = s[len(fs.separator):]
	}

	return s
}

func ensureNoPrefix(s string) string {
	if len(s) > 0 && strings.HasPrefix(s, gsPrefix) {
		return s[len(gsPrefix):]
	}
	return s
}

func validateName(s string) error {
	if len(s) == 0 {
		return ErrNoBucketInName
	}
	return nil
}

// Splits provided name into bucket name and path
func (fs *Fs) splitName(name string) (bucketName string, path string) {
	splitName := strings.Split(name, fs.separator)

	return splitName[0], strings.Join(splitName[1:], fs.separator)
}

// getBucket gets bucket info
//
func (fs *Fs) getBucket(name string) (*minio.BucketInfo, error) {
	bucket, is := fs.buckets[name]
	if !is {
		fs.setBucket(name)
		return fs.buckets[name], nil
	}
	return bucket, nil
}

func (fs *Fs) setBucket(name string) {
	fs.buckets[name] = &minio.BucketInfo{
		Name:         name,
		CreationDate: time.Now(),
	}
}

func (fs *Fs) getObj(name string) (*minio.Object, error) {
	bucketName, path := fs.splitName(name)
	getObjectOptions := minio.GetObjectOptions{}

	return fs.client.GetObject(fs.ctx, bucketName, path, getObjectOptions)
}

func (fs *Fs) Name() string { return "MinioFs" }

func (fs *Fs) Create(name string) (*MinioFile, error) {
	return fs.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0)
}

func (fs *Fs) Mkdir(name string, _ os.FileMode) error {
	// create bucket
	err := fs.client.MakeBucket(fs.ctx, name, minio.MakeBucketOptions{})
	if err != nil {
		return err
	}

	fs.setBucket(name)
	return nil
}

func (fs *Fs) MkdirAll(path string, perm os.FileMode) error {
	return errors.New("method MkdirAll is not implemented for Minio")
}

func (fs *Fs) Open(name string) (*MinioFile, error) {
	return fs.OpenFile(name, os.O_RDONLY, 0)
}

func (fs *Fs) OpenFile(name string, flag int, fileMode os.FileMode) (*MinioFile, error) {
	var file *MinioFile
	var err error

	name = fs.ensureNoLeadingSeparator(fs.normSeparators(ensureNoPrefix(name)))
	if err = validateName(name); err != nil {
		return nil, err
	}

	// folder creation logic has to additionally check for folder name presence
	bucketName, path := fs.splitName(name)
	bucket, err := fs.getBucket(bucketName)
	if err != nil {
		return nil, err
	}
	if path == "" {
		// the API would throw "Error 400: No object name, required", but this one is more consistent
		return nil, ErrEmptyObjectName
	}

	f, found := fs.rawMinioObjects[name]
	if found {
		file = NewMinioFileFromOldFH(flag, fileMode, f.resource)
	} else {
		file = NewMinioFile(fs.ctx, fs, bucket, flag, fileMode, path)
	}
	fs.rawMinioObjects[name] = file

	//if flag == os.O_RDONLY {
	//	_, err = file.Stat()
	//	if err != nil {
	//		return nil, err
	//	}
	//}
	//
	//if flag&os.O_TRUNC != 0 {
	//	err = file.resource.obj.Delete(fs.ctx)
	//	if err != nil {
	//		return nil, err
	//	}
	//	return fs.Create(name)
	//}
	//
	//if flag&os.O_APPEND != 0 {
	//	_, err = file.Seek(0, 2)
	//	if err != nil {
	//		return nil, err
	//	}
	//}
	//
	//if flag&os.O_CREATE != 0 {
	//	_, err = file.Stat()
	//	if err == nil { // the file actually exists
	//		return nil, syscall.EPERM
	//	}
	//
	//	_, err = file.WriteString("")
	//	if err != nil {
	//		return nil, err
	//	}
	//}

	return file, nil
}

func (fs *Fs) Remove(name string) error {
	name = fs.ensureNoLeadingSeparator(fs.normSeparators(ensureNoPrefix(name)))
	if err := validateName(name); err != nil {
		return err
	}

	bucketName, path := fs.splitName(name)

	return fs.client.RemoveObject(fs.ctx, bucketName, path, minio.RemoveObjectOptions{
		GovernanceBypass: true,
	})
}

func (fs *Fs) RemoveAll(path string) error {
	path = fs.ensureNoLeadingSeparator(fs.normSeparators(ensureNoPrefix(path)))
	if err := validateName(path); err != nil {
		return err
	}

	bucketName, dir := fs.splitName(path)
	objectsCh := make(chan minio.ObjectInfo)

	go func() {
		defer close(objectsCh)
		opts := minio.ListObjectsOptions{Prefix: dir, Recursive: true}
		for object := range fs.client.ListObjects(fs.ctx, bucketName, opts) {
			if object.Err != nil {
				log.Fatalln(object.Err)
			}
			objectsCh <- object
		}
	}()

	errorCh := fs.client.RemoveObjects(fs.ctx, bucketName, objectsCh, minio.RemoveObjectsOptions{})
	for e := range errorCh {
		return errors.New("Failed to remove " + e.ObjectName + ", error: " + e.Err.Error())
	}

	return nil
}

func (fs *Fs) Rename(oldName, newName string) error {
	oldName = fs.ensureNoLeadingSeparator(fs.normSeparators(ensureNoPrefix(oldName)))
	if err := validateName(oldName); err != nil {
		return err
	}

	newName = fs.ensureNoLeadingSeparator(fs.normSeparators(ensureNoPrefix(newName)))
	if err := validateName(newName); err != nil {
		return err
	}

	oldBucketName, oldPath := fs.splitName(oldName)
	newBucketName, newPath := fs.splitName(newName)

	// Source object
	src := minio.CopySrcOptions{
		Bucket: oldBucketName,
		Object: oldPath,
	}
	dst := minio.CopyDestOptions{
		Bucket: newBucketName,
		Object: newPath,
	}

	_, err := fs.client.CopyObject(fs.ctx, dst, src)
	if err != nil {
		return err
	}

	return fs.Remove(oldName)
}

func (fs *Fs) Stat(name string) (os.FileInfo, error) {
	name = fs.ensureNoLeadingSeparator(fs.normSeparators(ensureNoPrefix(name)))
	if err := validateName(name); err != nil {
		return nil, err
	}

	bucketName, path := fs.splitName(name)
	bucket, err := fs.getBucket(bucketName)
	if err != nil {
		return nil, err
	}

	file := NewMinioFile(fs.ctx, fs, bucket, os.O_RDWR, defaultFileMode, path)
	return file.Stat()
}

func (fs *Fs) Chmod(_ string, _ os.FileMode) error {
	return errors.New("method Chmod is not implemented in Minio")
}

func (fs *Fs) Chtimes(_ string, _, _ time.Time) error {
	return errors.New("method Chtimes is not implemented. Create, Delete, Updated times are read only fields in Minio and set implicitly")
}

func (fs *Fs) Chown(_ string, _, _ int) error {
	return errors.New("method Chown is not implemented for Minio")
}
