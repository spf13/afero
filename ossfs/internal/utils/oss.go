package utils

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/spf13/afero"
)

func init() {
	// Ensure OssObjectManager implements ObjectManager interface
	var _ ObjectManager = (*OssObjectManager)(nil)
}

var ossDirSeparator string = "/"
var ossDefaultFileMode fs.FileMode = 0o755

type OssObjectManager struct {
	ObjectManager
	Client *oss.Client
}

func (m *OssObjectManager) GetObject(ctx context.Context, bucket, name string) (io.Reader, CleanUp, error) {
	req := &oss.GetObjectRequest{
		Bucket: oss.Ptr(bucket),
		Key:    oss.Ptr(name),
	}
	res, err := m.Client.GetObject(ctx, req)
	cleanUp := func() {
		res.Body.Close()
	}
	return res.Body, cleanUp, err
}

func (m *OssObjectManager) GetObjectPart(ctx context.Context, bucket, name string, start, end int64) (io.Reader, CleanUp, error) {
	if start > end {
		return nil, nil, afero.ErrOutOfRange
	}
	req := &oss.GetObjectRequest{
		Bucket:        oss.Ptr(bucket),
		Key:           oss.Ptr(name),
		Range:         oss.Ptr(fmt.Sprintf("bytes=%v-%v", start, end)),
		RangeBehavior: oss.Ptr("standard"),
	}
	res, err := m.Client.GetObject(ctx, req)
	cleanUp := func() {
		res.Body.Close()
	}
	return res.Body, cleanUp, err
}

func (m *OssObjectManager) DeleteObject(ctx context.Context, bucket, name string) error {
	req := &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(bucket),
		Key:    oss.Ptr(name),
	}
	_, err := m.Client.DeleteObject(ctx, req)
	return err
}

func (m *OssObjectManager) IsObjectExist(ctx context.Context, bucket, name string) (bool, error) {
	return m.Client.IsObjectExist(ctx, bucket, name)
}

func (m *OssObjectManager) PutObject(ctx context.Context, bucket, name string, reader io.Reader) (bool, error) {
	req := &oss.PutObjectRequest{
		Bucket: oss.Ptr(bucket),
		Key:    oss.Ptr(name),
		Body:   reader,
	}
	_, err := m.Client.PutObject(ctx, req)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (m *OssObjectManager) CopyObject(ctx context.Context, bucket, srcName, targetName string) error {
	req := &oss.CopyObjectRequest{
		Bucket:       oss.Ptr(bucket),
		Key:          oss.Ptr(srcName),
		SourceKey:    oss.Ptr(targetName),
		SourceBucket: oss.Ptr(bucket),
		StorageClass: oss.StorageClassStandard,
	}
	_, err := m.Client.CopyObject(ctx, req)
	return err
}

func (m *OssObjectManager) GetObjectMeta(ctx context.Context, bucket, name string) (os.FileInfo, error) {
	req := &oss.HeadObjectRequest{
		Bucket: oss.Ptr(bucket),
		Key:    oss.Ptr(name),
	}

	res, err := m.Client.HeadObject(ctx, req)
	if err != nil {
		return nil, err
	}
	return &OssObjectMeta{
		name:           name,
		size:           res.ContentLength,
		lastModifiedAt: *res.LastModified,
	}, nil
}

func (m *OssObjectManager) ListObjects(ctx context.Context, bucket, prefix string, count int) ([]os.FileInfo, error) {
	req := &oss.ListObjectsV2Request{
		Bucket:    oss.Ptr(bucket),
		Delimiter: oss.Ptr(ossDirSeparator),
		Prefix:    oss.Ptr(prefix),
	}
	p := m.Client.NewListObjectsV2Paginator(req)

	s := make([]os.FileInfo, 0)

	var i int

loop:
	for p.HasNext() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, obj := range page.Contents {
			i++
			if i == count {
				break loop
			}
			s = append(s, &OssObjectMeta{
				name:           oss.ToString(obj.Key),
				size:           obj.Size,
				lastModifiedAt: oss.ToTime(obj.LastModified),
			})
		}
	}

	return s, nil
}

func (m *OssObjectManager) ListAllObjects(ctx context.Context, bucket, prefix string) ([]os.FileInfo, error) {
	req := &oss.ListObjectsV2Request{
		Bucket: oss.Ptr(bucket),
		Prefix: oss.Ptr(prefix),
	}
	p := m.Client.NewListObjectsV2Paginator(req)

	s := make([]os.FileInfo, 0)

	for p.HasNext() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, obj := range page.Contents {
			s = append(s, &OssObjectMeta{
				name:           oss.ToString(obj.Key),
				size:           obj.Size,
				lastModifiedAt: oss.ToTime(obj.LastModified),
			})
		}
	}

	return s, nil
}

type OssObjectMeta struct {
	os.FileInfo
	name           string
	size           int64
	lastModifiedAt time.Time
}

func NewOssObjectMeta(name string, size int64, updatedAt time.Time) *OssObjectMeta {
	return &OssObjectMeta{
		name:           name,
		size:           size,
		lastModifiedAt: updatedAt,
	}
}

func (objMeta *OssObjectMeta) isDir() bool {
	return strings.HasSuffix(objMeta.name, ossDirSeparator)
}

func (objMeta *OssObjectMeta) ModTime() time.Time {
	return objMeta.lastModifiedAt
}

func (objMeta *OssObjectMeta) Mode() fs.FileMode {
	if objMeta.isDir() {
		return ossDefaultFileMode | fs.ModeDir
	}
	return ossDefaultFileMode
}

func (objMeta *OssObjectMeta) Name() string {
	return objMeta.name
}

func (objMeta *OssObjectMeta) Size() int64 {
	return objMeta.size
}

func (objMeta *OssObjectMeta) Sys() any {
	return nil
}
