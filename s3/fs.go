package s3

import (
	"errors"
	"os"
	"time"

	minio "github.com/minio/minio-go"
	"github.com/spf13/afero"
)

func New(client *minio.Client, bucketName string) afero.Fs {
	return &Fs{
		client:     client,
		bucketName: bucketName,
	}
}

type Fs struct {
	client     *minio.Client
	bucketName string
}

func (f *Fs) Name() string {
	return "s3fs"
}

func (f *Fs) Create(name string) (afero.File, error) {
	return FileCreate(f.client, f.bucketName, name, "binary/octet-stream")
}

func (f *Fs) Mkdir(name string, perm os.FileMode) error {
	//TODO: Missing in afero/s3
	return errors.New("TODO/missing in afero/s3")
}

func (f *Fs) MkdirAll(path string, perm os.FileMode) error {
	//TODO: Missing in afero/s3
	return errors.New("TODO/missing in afero/s3")
}

func (f *Fs) Open(name string) (afero.File, error) {
	return FileOpen(f.client, f.bucketName, name)
}

func (f *Fs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	//TODO: Missing in afero/s3, not using flag or perm
	return FileOpen(f.client, f.bucketName, name)
}

func (f *Fs) Remove(name string) error {
	return f.client.RemoveObject(f.bucketName, name)
}

func (f *Fs) RemoveAll(path string) error {
	//TODO: Missing in afero/s3
	return errors.New("TODO/missing in afero/s3")
}

func (f *Fs) Rename(oldname, newname string) error {
	//TODO: Missing in afero/s3
	return errors.New("TODO/missing in afero/s3")
}

func (f *Fs) Stat(name string) (os.FileInfo, error) {
	objStat, err := f.client.StatObject(f.bucketName, name)
	if err != nil {
		return nil, err
	}
	return NewFileInfo(name, objStat), nil
}

func (f *Fs) Chmod(name string, mode os.FileMode) error {
	//TODO: Missing in afero/s3
	return nil
}

func (f *Fs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	//TODO: Missing in afero/s3
	return nil
}
