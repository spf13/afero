package minio

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/afero"
	"log"
	"os"
	"path/filepath"
	"time"
)

type MinioFs struct {
	bucket string
	source *Fs
}

func NewMinio(ctx context.Context, endpoint string, options *minio.Options) *MinioFs {
	return NewMinioWithBucket(ctx, endpoint, "", options)
}

func NewMinioWithBucket(ctx context.Context, endpoint string, bucket string, options *minio.Options) *MinioFs {
	s3Client, err := minio.New(endpoint, options)
	if err != nil {
		log.Fatal("minio file initialization failed, err: ", err)
		return nil
	}

	minioFs := NewMinioFs(ctx, s3Client)
	return &MinioFs{
		bucket: bucket,
		source: minioFs,
	}
}

func (fs *MinioFs) Name() string {
	return fs.source.Name()
}

func (fs *MinioFs) Create(name string) (afero.File, error) {
	return fs.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0)
}

func (fs *MinioFs) Mkdir(name string, perm os.FileMode) error {
	return fs.source.Mkdir(name, perm)
}

func (fs *MinioFs) MkdirAll(path string, perm os.FileMode) error {
	return fs.source.MkdirAll(path, perm)
}

func (fs *MinioFs) Open(name string) (afero.File, error) {
	return fs.OpenFile(name, os.O_RDONLY, 0)
}

func (fs *MinioFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if fs.bucket != "" {
		name = filepath.Join(fs.bucket, name)
	}
	return fs.source.OpenFile(name, flag, perm)
}

func (fs *MinioFs) Remove(name string) error {
	if fs.bucket != "" {
		name = filepath.Join(fs.bucket, name)
	}
	return fs.source.Remove(name)
}

func (fs *MinioFs) RemoveAll(path string) error {
	if fs.bucket != "" {
		path = filepath.Join(fs.bucket, path)
	}
	return fs.source.RemoveAll(path)
}

func (fs *MinioFs) Rename(oldname, newname string) error {
	if fs.bucket != "" {
		oldname = filepath.Join(fs.bucket, oldname)
		newname = filepath.Join(fs.bucket, newname)
	}
	return fs.source.Rename(oldname, newname)
}

func (fs *MinioFs) Stat(name string) (os.FileInfo, error) {
	if fs.bucket != "" {
		name = filepath.Join(fs.bucket, name)
	}
	return fs.source.Stat(name)
}

func (fs *MinioFs) Chmod(name string, mode os.FileMode) error {
	if fs.bucket != "" {
		name = filepath.Join(fs.bucket, name)
	}
	return fs.source.Chmod(name, mode)
}

func (fs *MinioFs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	if fs.bucket != "" {
		name = filepath.Join(fs.bucket, name)
	}
	return fs.source.Chtimes(name, atime, mtime)
}

func (fs *MinioFs) Chown(name string, uid, gid int) error {
	if fs.bucket != "" {
		name = filepath.Join(fs.bucket, name)
	}
	return fs.source.Chown(name, uid, gid)
}
