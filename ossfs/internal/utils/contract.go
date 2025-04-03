package utils

import (
	"io"
	"os"
)

type ObjectManager interface {
	GetObject(bucket, name string) (io.Reader, error)
	IsObjectExist(bucket, name string) (bool, error)
	PutObject(bucket, name string, reader io.Reader) (bool, error)
	GetObjectMeta(bucket, name string) (ObjectMeta, error)
	ListObjects(bucket, prefix string) ([]ObjectMeta, error)
	ListAllObjects(bucket, prefix string) ([]ObjectMeta, error)
}

type ObjectMeta os.FileInfo
