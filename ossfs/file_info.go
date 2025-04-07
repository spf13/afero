package ossfs

import (
	"time"

	"github.com/spf13/afero/ossfs/internal/utils"
)

type FileInfo struct {
	*utils.OssObjectMeta
}

func NewFileInfo(name string, size int64, updatedAt time.Time) *FileInfo {
	return &FileInfo{
		OssObjectMeta: utils.NewOssObjectMeta(name, size, updatedAt),
	}
}
