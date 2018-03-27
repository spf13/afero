// Copyright © 2018 Mikael Rapp, github.com/zatte
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

package afero

import (
	"log"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"github.com/zatte/afero/gcs"

	"golang.org/x/net/context"
)

func NewGcsFs(ctx context.Context, cl *storage.Client, bucket string, folderSep string) Fs {
	return &GcsFsWrapper{
		gcs.NewGcsFs(ctx, cl, bucket, folderSep),
	}
}

// NewGcsFsFromDefaultCredentials Creates a GCS client assuming that
// $GOOGLE_APPLICATION_CREDENTIALS is set and points to a service account
func NewGcsFsFromDefaultCredentials(ctx context.Context, bucket string, folderSep string) Fs {
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	return &GcsFsWrapper{
		gcs.NewGcsFs(ctx, client, bucket, folderSep),
	}
}

//Only wrapes gcs.GcsFs and convert some return types to afero interfaces.
type GcsFsWrapper struct {
	GcsFs *gcs.GcsFs
}

func (fs *GcsFsWrapper) Name() string {
	return fs.GcsFs.Name()
}
func (fs *GcsFsWrapper) Create(name string) (File, error) {
	return fs.GcsFs.Create(name)
}
func (fs *GcsFsWrapper) Mkdir(name string, perm os.FileMode) error {
	return fs.GcsFs.Mkdir(name, perm)
}
func (fs *GcsFsWrapper) MkdirAll(path string, perm os.FileMode) error {
	return fs.GcsFs.MkdirAll(path, perm)
}
func (fs *GcsFsWrapper) Open(name string) (File, error) {
	return fs.GcsFs.Open(name)
}
func (fs *GcsFsWrapper) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	return fs.GcsFs.OpenFile(name, flag, perm)
}
func (fs *GcsFsWrapper) Remove(name string) error {
	return fs.GcsFs.Remove(name)
}
func (fs *GcsFsWrapper) RemoveAll(path string) error {
	return fs.GcsFs.RemoveAll(path)
}
func (fs *GcsFsWrapper) Rename(oldname, newname string) error {
	return fs.GcsFs.Rename(oldname, newname)
}
func (fs *GcsFsWrapper) Stat(name string) (os.FileInfo, error) {
	return fs.GcsFs.Stat(name)
}
func (fs *GcsFsWrapper) Chmod(name string, mode os.FileMode) error {
	return fs.GcsFs.Chmod(name, mode)
}
func (fs *GcsFsWrapper) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return fs.GcsFs.Chtimes(name, atime, mtime)
}
