package s3

import (
	"os"
	"time"
)

// S3FileInfo implements os.FileInfo for a file in S3.
type S3FileInfo struct {
	name        string
	directory   bool
	sizeInBytes int64
	modTime     time.Time
}

func NewS3FileInfo(name string, directory bool, sizeInBytes int64, modTime time.Time) S3FileInfo {
	return S3FileInfo{
		name:        name,
		directory:   directory,
		sizeInBytes: sizeInBytes,
	}
}

var _ os.FileInfo = (*S3FileInfo)(nil)

// Name provides the base name of the file.
func (fi S3FileInfo) Name() string {
	return fi.name
}

// Size provides the length in bytes for a file.
func (fi S3FileInfo) Size() int64 {
	return fi.sizeInBytes
}

// Mode provides the file mode bits. For a file in S3 this defaults to
// 664 for files, 775 for directories.
// In the future this may return differently depending on the permissions
// available on the bucket.
func (fi S3FileInfo) Mode() os.FileMode {
	if fi.directory {
		return 0755
	}
	return 0664
}

// ModTime provides the last modification time.
func (fi S3FileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir provides the abbreviation for Mode().IsDir()
func (fi S3FileInfo) IsDir() bool {
	return fi.directory
}

// Sys provides the underlying data source (can return nil)
func (fi S3FileInfo) Sys() interface{} {
	return nil
}
