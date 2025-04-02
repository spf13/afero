package ossfs

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
)

type FileInfo struct {
	fs        *Fs
	name      string
	size      int64
	updatedAt time.Time
	isDir     bool
	fileMode  os.FileMode
}

func NewFileInfo(name string, fs *Fs, res *oss.HeadObjectResult) *FileInfo {
	return &FileInfo{
		fs:        fs,
		name:      name,
		size:      res.ContentLength,
		updatedAt: oss.ToTime(res.LastModified),
		isDir:     strings.HasSuffix(name, fs.separator),
		fileMode:  defaultFileMode,
	}
}

func NewFileInfoWithObjProp(name string, fs *Fs, props oss.ObjectProperties) *FileInfo {
	return &FileInfo{
		fs:        fs,
		name:      name,
		size:      props.Size,
		updatedAt: oss.ToTime(props.LastModified),
		isDir:     fs.isDir(name),
		fileMode:  defaultFileMode,
	}
}

func (fi *FileInfo) Name() string {
	return filepath.Base(filepath.FromSlash(fi.name))
}

func (fi *FileInfo) Size() int64 {
	return fi.size
}

func (fi *FileInfo) Mode() os.FileMode {
	if fi.IsDir() {
		return os.ModeDir | fi.fileMode
	}
	return fi.fileMode
}

func (fi *FileInfo) ModTime() time.Time {
	return fi.updatedAt
}

func (fi *FileInfo) IsDir() bool {
	return fi.isDir
}

func (fi *FileInfo) Sys() any {
	return nil
}
