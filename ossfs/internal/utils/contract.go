package utils

import (
	"context"
	"io"
	"os"
)

type CleanUp func()

type ObjectManager interface {
	GetObject(ctx context.Context, bucket, name string) (io.Reader, CleanUp, error)
	GetObjectPart(ctx context.Context, bucket, name string, start, end int64) (io.Reader, CleanUp, error)
	DeleteObject(ctx context.Context, bucket, name string) error
	IsObjectExist(ctx context.Context, bucket, name string) (bool, error)
	PutObject(ctx context.Context, bucket, name string, reader io.Reader) (bool, error)
	CopyObject(ctx context.Context, bucket, srcName, targetName string) error
	GetObjectMeta(ctx context.Context, bucket, name string) (os.FileInfo, error)
	ListObjects(ctx context.Context, bucket, prefix string, count int) ([]os.FileInfo, error)
	ListAllObjects(ctx context.Context, bucket, prefix string) ([]os.FileInfo, error)
}
