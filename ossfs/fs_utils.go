package ossfs

import (
	"io"
	"strings"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
)

func (fs *Fs) isDir(s string) bool {
	return strings.HasSuffix(s, fs.separator)
}

func (fs *Fs) ensureAsDir(s string) string {
	return fs.ensureTrailingSeparator(fs.normSeparators(s))
}

func (fs *Fs) normSeparators(s string) string {
	return strings.Replace(strings.Replace(s, "\\", fs.separator, -1), "/", fs.separator, -1)
}

func (fs *Fs) ensureTrailingSeparator(s string) string {
	if len(s) > 0 && !fs.isDir(s) {
		return s + fs.separator
	}
	return s
}

func (fs *Fs) putObjectStr(name, str string) (*oss.PutObjectResult, error) {
	return fs.putObjectReader(name, strings.NewReader(str))
}

func (fs *Fs) putObjectReader(name string, reader io.Reader) (*oss.PutObjectResult, error) {
	req := &oss.PutObjectRequest{
		Bucket: oss.Ptr(fs.bucketName),
		Key:    oss.Ptr(name),
		Body:   reader,
	}
	return fs.client.PutObject(fs.ctx, req)
}

func (fs *Fs) deleteObject(name string) (*oss.DeleteObjectResult, error) {
	req := &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(fs.bucketName),
		Key:    oss.Ptr(name),
	}
	return fs.client.DeleteObject(fs.ctx, req)
}

// Rename renames a file.
func (fs *Fs) copyObject(oldname, newname string) (*oss.CopyObjectResult, error) {
	req := &oss.CopyObjectRequest{
		Bucket:       oss.Ptr(fs.bucketName),
		Key:          oss.Ptr(newname),
		SourceKey:    oss.Ptr(oldname),
		SourceBucket: oss.Ptr(fs.bucketName),
		StorageClass: oss.StorageClassStandard,
	}
	return fs.client.CopyObject(fs.ctx, req)
}

func (fs *Fs) isObjectExists(name string) (bool, error) {
	return fs.client.IsObjectExist(fs.ctx, fs.bucketName, name)
}
