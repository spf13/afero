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

package afero

import (
	"context"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googleapis/google-cloud-go-testing/storage/stiface"
	"github.com/spf13/afero/gcsfs"
	"google.golang.org/api/option"
)

type GcsFs struct {
	source *gcsfs.GcsFs
}

// Creates a GCS file system, automatically instantiating and decorating the storage client.
// You can provide additional options to be passed to the client creation, as per
// cloud.google.com/go/storage documentation
func NewGcsFS(ctx context.Context, bucketName string, opts ...option.ClientOption) (Fs, error) {
	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return NewGcsFSFromClient(ctx, client, bucketName)
}

// The same as NewGcsFS, but the files system will use the provided folder separator.
func NewGcsFSWithSeparator(ctx context.Context, bucketName, folderSeparator string, opts ...option.ClientOption) (Fs, error) {
	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return NewGcsFSFromClientWithSeparator(ctx, client, bucketName, folderSeparator)
}

// Creates a GCS file system from a given storage client
func NewGcsFSFromClient(ctx context.Context, client *storage.Client, bucketName string) (Fs, error) {
	c := stiface.AdaptClient(client)

	bucket := c.Bucket(bucketName)

	return &GcsFs{gcsfs.NewGcsFs(ctx, bucket)}, nil
}

// Same as NewGcsFSFromClient, but the file system will use the provided folder separator.
func NewGcsFSFromClientWithSeparator(ctx context.Context, client *storage.Client, bucketName, folderSeparator string) (Fs, error) {
	c := stiface.AdaptClient(client)

	bucket := c.Bucket(bucketName)

	return &GcsFs{gcsfs.NewGcsFsWithSeparator(ctx, bucket, folderSeparator)}, nil
}

// Wraps gcs.GcsFs and convert some return types to afero interfaces.
func (fs *GcsFs) Name() string {
	return fs.source.Name()
}
func (fs *GcsFs) Create(name string) (File, error) {
	return fs.source.Create(name)
}
func (fs *GcsFs) Mkdir(name string, perm os.FileMode) error {
	return fs.source.Mkdir(name, perm)
}
func (fs *GcsFs) MkdirAll(path string, perm os.FileMode) error {
	return fs.source.MkdirAll(path, perm)
}
func (fs *GcsFs) Open(name string) (File, error) {
	return fs.source.Open(name)
}
func (fs *GcsFs) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	return fs.source.OpenFile(name, flag, perm)
}
func (fs *GcsFs) Remove(name string) error {
	return fs.source.Remove(name)
}
func (fs *GcsFs) RemoveAll(path string) error {
	return fs.source.RemoveAll(path)
}
func (fs *GcsFs) Rename(oldname, newname string) error {
	return fs.source.Rename(oldname, newname)
}
func (fs *GcsFs) Stat(name string) (os.FileInfo, error) {
	return fs.source.Stat(name)
}
func (fs *GcsFs) Chmod(name string, mode os.FileMode) error {
	return fs.source.Chmod(name, mode)
}
func (fs *GcsFs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return fs.source.Chtimes(name, atime, mtime)
}
func (fs *GcsFs) Chown(name string, uid, gid int) error {
	return fs.source.Chown(name, uid, gid)
}
