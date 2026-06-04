// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stiface

import (
	"context"
	"io"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

type Client interface {
	Bucket(name string) BucketHandle
	Buckets(ctx context.Context, projectID string) BucketIterator
	Close() error

	embedToIncludeNewMethods()
}

type ObjectHandle interface {
	ACL() ACLHandle
	Generation(int64) ObjectHandle
	If(storage.Conditions) ObjectHandle
	Key([]byte) ObjectHandle
	ReadCompressed(bool) ObjectHandle
	Attrs(context.Context) (*storage.ObjectAttrs, error)
	Update(context.Context, storage.ObjectAttrsToUpdate) (*storage.ObjectAttrs, error)
	NewReader(context.Context) (Reader, error)
	NewRangeReader(context.Context, int64, int64) (Reader, error)
	NewWriter(context.Context) Writer
	Delete(context.Context) error
	CopierFrom(ObjectHandle) Copier
	ComposerFrom(...ObjectHandle) Composer

	embedToIncludeNewMethods()
}

type BucketHandle interface {
	Create(context.Context, string, *storage.BucketAttrs) error
	Delete(context.Context) error
	DefaultObjectACL() ACLHandle
	Object(string) ObjectHandle
	Attrs(context.Context) (*storage.BucketAttrs, error)
	Update(context.Context, storage.BucketAttrsToUpdate) (*storage.BucketAttrs, error)
	If(storage.BucketConditions) BucketHandle
	Objects(context.Context, *storage.Query) ObjectIterator
	ACL() ACLHandle
	// IAM() *iam.Handle
	UserProject(projectID string) BucketHandle
	Notifications(context.Context) (map[string]*storage.Notification, error)
	AddNotification(context.Context, *storage.Notification) (*storage.Notification, error)
	DeleteNotification(context.Context, string) error
	LockRetentionPolicy(context.Context) error

	embedToIncludeNewMethods()
}

type ObjectIterator interface {
	Next() (*storage.ObjectAttrs, error)
	PageInfo() *iterator.PageInfo

	embedToIncludeNewMethods()
}

type BucketIterator interface {
	SetPrefix(string)
	Next() (*storage.BucketAttrs, error)
	PageInfo() *iterator.PageInfo

	embedToIncludeNewMethods()
}

type ACLHandle interface {
	Delete(context.Context, storage.ACLEntity) error
	Set(context.Context, storage.ACLEntity, storage.ACLRole) error
	List(context.Context) ([]storage.ACLRule, error)

	embedToIncludeNewMethods()
}

type Reader interface {
	io.ReadCloser
	Size() int64
	Remain() int64
	ContentType() string
	ContentEncoding() string
	CacheControl() string

	embedToIncludeNewMethods()
}

type Writer interface {
	io.WriteCloser
	ObjectAttrs() *storage.ObjectAttrs
	SetChunkSize(int)
	SetProgressFunc(func(int64))
	SetCRC32C(uint32) // Sets both CRC32C and SendCRC32C.
	CloseWithError(err error) error
	Attrs() *storage.ObjectAttrs

	embedToIncludeNewMethods()
}

type Copier interface {
	ObjectAttrs() *storage.ObjectAttrs
	SetRewriteToken(string)
	SetProgressFunc(func(uint64, uint64))
	SetDestinationKMSKeyName(string)
	Run(context.Context) (*storage.ObjectAttrs, error)

	embedToIncludeNewMethods()
}

type Composer interface {
	ObjectAttrs() *storage.ObjectAttrs
	Run(context.Context) (*storage.ObjectAttrs, error)

	embedToIncludeNewMethods()
}
