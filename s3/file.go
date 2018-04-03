package s3

import (
	"bytes"
	"errors"
	"net/http"
	"os"

	minio "github.com/minio/minio-go"
)

func FileCreate(c *minio.Client, bucketName, name, contentType string) (*File, error) {
	_, err := c.PutObject(bucketName, name, bytes.NewReader([]byte{}), contentType)
	if err != nil {
		return nil, err
	}
	return &File{
		c:           c,
		bucket:      bucketName,
		name:        name,
		contentType: contentType,
	}, nil
}

func FileOpen(c *minio.Client, bucketName, name string) (*File, error) {
	stat, err := c.StatObject(bucketName, name)
	if err != nil {
		return nil, err
	}
	return &File{
		c:           c,
		bucket:      bucketName,
		name:        name,
		contentType: stat.ContentType,
	}, nil
}

type File struct {
	c           *minio.Client
	bucket      string
	name        string
	contentType string
}

func (f *File) Name() string {
	return f.name
}

func (f *File) Close() error {
	return nil
}

func (f *File) Read(p []byte) (n int, err error) {
	obj, err := f.c.GetObject(f.bucket, f.name)
	if err != nil {
		return 0, err
	}
	return obj.Read(p)
}

func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	obj, err := f.c.GetObject(f.bucket, f.name)
	if err != nil {
		return 0, err
	}
	return obj.ReadAt(p, off)
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	obj, err := f.c.GetObject(f.bucket, f.name)
	if err != nil {
		return 0, err
	}
	return obj.Seek(offset, whence)
}

func (f *File) Write(p []byte) (n int, err error) {
	contentType := http.DetectContentType(p)
	f.contentType = contentType
	reader := bytes.NewReader(p)
	i, err := f.c.PutObject(f.bucket, f.name, reader, contentType)
	return int(i), err
}

func (f *File) WriteAt(p []byte, off int64) (n int, err error) {
	//TODO: Missing in afero/s3
	return 0, errors.New("TODO/missing in afero/s3")
}

func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	//TODO: Missing in afero/s3
	return nil, errors.New("TODO/missing in afero/s3")
}

func (f *File) Readdirnames(n int) ([]string, error) {
	//TODO: Missing in afero/s3
	return nil, errors.New("TODO/missing in afero/s3")
}

func (f *File) Stat() (os.FileInfo, error) {
	obj, err := f.c.GetObject(f.bucket, f.name)
	if err != nil {
		return nil, err
	}
	info, err := obj.Stat()
	if err != nil {
		return nil, err
	}
	return NewFileInfo(f.name, info), nil
}

func (f *File) Sync() error {
	return nil
}

func (f *File) Truncate(size int64) error {
	//TODO: Missing in afero/s3
	return errors.New("TODO/missing in afero/s3")
}

func (f *File) WriteString(s string) (ret int, err error) {
	return f.Write([]byte(s))
}
