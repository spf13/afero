// Copyright © 2021 Vasily Ovchinnikov <vasily@remerge.io>.
//
// The code in this file is derived from afero fork github.com/Zatte/afero by Mikael Rapp
// licensed under Apache License 2.0.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gcsfs

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/googleapis/google-cloud-go-testing/storage/stiface"
)

const (
	defaultFileMode = 0755
)

// GcsFs is a Fs implementation that uses functions provided by google cloud storage
type GcsFs struct {
	ctx           context.Context
	bucket        stiface.BucketHandle
	separator     string
	rawGcsObjects map[string]*GcsFile

	autoRemoveEmptyFolders bool //trigger for creating "virtual folders" (not required by GCSs)
}

func NewGcsFs(ctx context.Context, bucket stiface.BucketHandle) *GcsFs {
	return NewGcsFsWithSeparator(ctx, bucket, "/")
}

func NewGcsFsWithSeparator(ctx context.Context, bucket stiface.BucketHandle, folderSep string) *GcsFs {
	return &GcsFs{
		ctx:           ctx,
		bucket:        bucket,
		separator:     folderSep,
		rawGcsObjects: make(map[string]*GcsFile),

		autoRemoveEmptyFolders: true,
	}
}

// normSeparators will normalize all "\\" and "/" to the provided separator
func (fs *GcsFs) normSeparators(s string) string {
	return strings.Replace(strings.Replace(s, "\\", fs.separator, -1), "/", fs.separator, -1)
}

func (fs *GcsFs) ensureTrailingSeparator(s string) string {
	if len(s) > 0 && !strings.HasSuffix(s, fs.separator) {
		return s + fs.separator
	}
	return s
}

func (fs *GcsFs) ensureNoLeadingSeparators(s string) string {
	// GCS does REALLY not like the names, that begin with a separator
	if len(s) > 0 && strings.HasPrefix(s, fs.separator) {
		log.Printf(
			"WARNING: the provided path \"%s\" starts with a separator \"%s\", which is not supported by "+
				"GCloud. The separator will be automatically trimmed",
			s,
			fs.separator,
		)
		return s[len(fs.separator):]
	}
	return s
}

func correctTheDot(s string) string {
	// So, Afero's Glob likes to give "." as a name - that to list the "empty" dir name.
	// GCS _really_ dislikes the dot and gives no entries for it - so we should rather replace the dot
	// with an empty string
	if s == "." {
		return ""
	}
	return s
}

func (fs *GcsFs) getObj(name string) stiface.ObjectHandle {
	return fs.bucket.Object(name)
}

func (fs *GcsFs) Name() string { return "GcsFs" }

func (fs *GcsFs) Create(name string) (*GcsFile, error) {
	name = fs.ensureNoLeadingSeparators(fs.normSeparators(correctTheDot(name)))

	if !fs.autoRemoveEmptyFolders {
		baseDir := filepath.Base(name)
		if stat, err := fs.Stat(baseDir); err != nil || !stat.IsDir() {
			err = fs.MkdirAll(baseDir, 0)
			if err != nil {
				return nil, err
			}
		}
	}

	obj := fs.getObj(name)
	w := obj.NewWriter(fs.ctx)
	var err error
	err = w.Close()
	if err != nil {
		return nil, err
	}
	file := NewGcsFile(fs.ctx, fs, obj, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0, name)

	fs.rawGcsObjects[name] = file
	return file, nil
}

func (fs *GcsFs) Mkdir(name string, _ os.FileMode) error {
	name = fs.ensureTrailingSeparator(fs.ensureNoLeadingSeparators(fs.normSeparators(correctTheDot(name))))

	obj := fs.getObj(name)
	w := obj.NewWriter(fs.ctx)
	return w.Close()
}

func (fs *GcsFs) MkdirAll(path string, perm os.FileMode) error {
	path = fs.ensureTrailingSeparator(fs.ensureNoLeadingSeparators(fs.normSeparators(correctTheDot(path))))

	root := ""
	folders := strings.Split(path, fs.separator)
	for i, f := range folders {
		if f == "" && i != 0 {
			continue // it's the last item - it should be empty
		}
		//Don't force a delimiter prefix
		if root != "" {
			root = root + fs.separator + f
		} else {
			root = f
		}

		if err := fs.Mkdir(root, perm); err != nil {
			return err
		}
	}
	return nil
}

func (fs *GcsFs) Open(name string) (*GcsFile, error) {
	return fs.OpenFile(name, os.O_RDONLY, 0)
}

func (fs *GcsFs) OpenFile(name string, flag int, fileMode os.FileMode) (*GcsFile, error) {
	var file *GcsFile
	var err error

	name = fs.ensureNoLeadingSeparators(fs.normSeparators(correctTheDot(name)))

	obj, found := fs.rawGcsObjects[name]
	if found {
		file = NewGcsFileFromOldFH(flag, fileMode, obj.resource)
	} else {
		file = NewGcsFile(fs.ctx, fs, fs.getObj(name), flag, fileMode, name)
	}

	if flag == os.O_RDONLY {
		_, err = file.Stat()
		if err != nil {
			return nil, err
		}
	}

	if flag&os.O_TRUNC != 0 {
		err = file.resource.obj.Delete(fs.ctx)
		if err != nil {
			return nil, err
		}
		return fs.Create(name)
	}

	if flag&os.O_APPEND != 0 {
		_, err = file.Seek(0, 2)
		if err != nil {
			return nil, err
		}
	}

	if flag&os.O_CREATE != 0 {
		_, err = file.Stat()
		if err == nil { // the file actually exists
			return nil, syscall.EPERM
		}

		_, err = file.WriteString("")
		if err != nil {
			return nil, err
		}
	}
	return file, nil
}

func (fs *GcsFs) Remove(name string) error {
	name = fs.ensureNoLeadingSeparators(fs.normSeparators(correctTheDot(name)))

	obj := fs.getObj(name)
	info, err := fs.Stat(name)
	if err != nil {
		return err
	}
	delete(fs.rawGcsObjects, name)

	if info.IsDir() {
		// it's a folder, we ha to check its contents - it cannot be removed, if not empty
		var dir *GcsFile
		dir, err = fs.Open(name)
		if err != nil {
			return err
		}
		var infos []os.FileInfo
		infos, err = dir.Readdir(0)
		if len(infos) > 0 {
			return syscall.ENOTEMPTY
		}

		// it's an empty folder, we can continue
		name = fs.ensureTrailingSeparator(name)
		obj = fs.getObj(name)

		return obj.Delete(fs.ctx)
	}
	return obj.Delete(fs.ctx)
}

func (fs *GcsFs) RemoveAll(path string) error {
	path = fs.ensureNoLeadingSeparators(fs.normSeparators(correctTheDot(path)))

	pathInfo, err := fs.Stat(path)
	if err != nil {
		return err
	}
	if !pathInfo.IsDir() {
		return fs.Remove(path)
	}

	var dir *GcsFile
	dir, err = fs.Open(path)
	if err != nil {
		return err
	}

	var infos []os.FileInfo
	infos, err = dir.Readdir(0)
	for _, info := range infos {
		err = fs.RemoveAll(path + fs.separator + info.Name())
		if err != nil {
			return err
		}
	}

	return fs.Remove(path)

	//it := fs.bucket.Objects(fs.ctx, &storage.Query{Delimiter: fs.separator, Prefix: path, Versions: false})
	//for {
	//	objAttrs, err := it.Next()
	//	if err == iterator.Done {
	//		break
	//	}
	//	if err != nil {
	//		return err
	//	}
	//
	//	name := objAttrs.Name
	//	if name == "" {
	//		name = objAttrs.Prefix
	//	}
	//
	//	if name == path {
	//		// somehow happens
	//		continue
	//	}
	//	if objAttrs.Name == "" && objAttrs.Prefix != "" {
	//		// it's a folder, let's try to remove it normally first
	//		err = fs.Remove(path + fs.separator + objAttrs.Name)
	//		if err != nil {
	//			if err == syscall.ENOTEMPTY {
	//				err = fs.RemoveAll(path + fs.separator + objAttrs.Name)
	//			}
	//		}
	//		if err != nil {
	//			return err
	//		}
	//
	//	} else {
	//		err = fs.Remove(objAttrs.Name)
	//		if err != nil {
	//			return err
	//		}
	//	}
	//}
	//return nil
}

func (fs *GcsFs) Rename(oldName, newName string) error {
	oldName = fs.ensureNoLeadingSeparators(fs.normSeparators(correctTheDot(oldName)))
	newName = fs.ensureNoLeadingSeparators(fs.normSeparators(correctTheDot(newName)))

	src := fs.bucket.Object(oldName)
	dst := fs.bucket.Object(newName)

	if _, err := dst.CopierFrom(src).Run(fs.ctx); err != nil {
		return err
	}
	delete(fs.rawGcsObjects, oldName)
	return src.Delete(fs.ctx)
}

func (fs *GcsFs) Stat(name string) (os.FileInfo, error) {
	name = fs.ensureNoLeadingSeparators(fs.normSeparators(correctTheDot(name)))

	return newFileInfo(name, fs, defaultFileMode)
}

func (fs *GcsFs) Chmod(_ string, _ os.FileMode) error {
	return errors.New("method Chmod is not implemented in GCS")
}

func (fs *GcsFs) Chtimes(_ string, _, _ time.Time) error {
	return errors.New("method Chtimes is not implemented. Create, Delete, Updated times are read only fields in GCS and set implicitly")
}

func (fs *GcsFs) Chown(_ string, _, _ int) error {
	return errors.New("method Chown is not implemented for GCS")
}
