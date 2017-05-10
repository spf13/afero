package s3

import (
	"os"
	"time"

	minio "github.com/minio/minio-go"
)

func NewFileInfo(name string, o minio.ObjectInfo) os.FileInfo {
	return &fileInfo{
		o:    o,
		name: name,
	}
}

type fileInfo struct {
	o    minio.ObjectInfo
	name string
}

func (f *fileInfo) Name() string {
	return f.name
}

func (f *fileInfo) Size() int64 {
	return f.o.Size
}

func (f *fileInfo) Mode() os.FileMode {
	//TODO: Missing in afero/s3
	return 0
}

func (f *fileInfo) ModTime() time.Time {
	return f.o.LastModified
}

func (f *fileInfo) IsDir() bool {
	//TODO: Missing in afero/s3
	return false
}

func (f *fileInfo) Sys() interface{} {
	return f.o
}
