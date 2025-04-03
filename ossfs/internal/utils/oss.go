package utils

import (
	"context"
	"io"
	"io/fs"
	"strings"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
)

var ossDirSeparator string = "/"
var ossDefaultFileMode fs.FileMode = 0o755

type OssObjectManager struct {
	ObjectManager
	Ctx    context.Context
	Client *oss.Client
}

func (m *OssObjectManager) GetObject(bucket, name string) (io.Reader, error) {
	req := &oss.GetObjectRequest{
		Bucket: oss.Ptr(bucket),
		Key:    oss.Ptr(name),
	}
	res, err := m.Client.GetObject(m.Ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Body, err
}

func (m *OssObjectManager) IsObjectExist(bucket, name string) (bool, error) {
	return m.Client.IsObjectExist(m.Ctx, bucket, name)
}

func (m *OssObjectManager) PutObject(bucket, name string, reader io.Reader) (bool, error) {
	req := &oss.PutObjectRequest{
		Bucket: oss.Ptr(bucket),
		Key:    oss.Ptr(name),
		Body:   reader,
	}
	_, err := m.Client.PutObject(m.Ctx, req)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (m *OssObjectManager) GetObjectMeta(bucket, name string) (ObjectMeta, error) {
	req := &oss.HeadObjectRequest{
		Bucket: oss.Ptr(bucket),
		Key:    oss.Ptr(name),
	}

	res, err := m.Client.HeadObject(m.Ctx, req)
	if err != nil {
		return nil, err
	}
	return &OssObjectMeta{
		name:           name,
		size:           res.ContentLength,
		lastModifiedAt: *res.LastModified,
	}, nil
}

func (m *OssObjectManager) ListObjects(bucket, prefix string) ([]ObjectMeta, error) {
	req := &oss.ListObjectsV2Request{
		Bucket:    oss.Ptr(bucket),
		Delimiter: oss.Ptr(ossDirSeparator),
		Prefix:    oss.Ptr(prefix),
	}
	p := m.Client.NewListObjectsV2Paginator(req)

	s := make([]ObjectMeta, 0)

	for p.HasNext() {
		page, err := p.NextPage(m.Ctx)
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

func (m *OssObjectManager) ListAllObjects(bucket, prefix string) ([]ObjectMeta, error) {
	req := &oss.ListObjectsV2Request{
		Bucket: oss.Ptr(bucket),
		Prefix: oss.Ptr(prefix),
	}
	p := m.Client.NewListObjectsV2Paginator(req)

	s := make([]ObjectMeta, 0)

	for p.HasNext() {
		page, err := p.NextPage(m.Ctx)
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
	ObjectMeta
	name           string
	size           int64
	lastModifiedAt time.Time
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
