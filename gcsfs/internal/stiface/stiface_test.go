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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"cloud.google.com/go/storage"
)

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration tests skipped in short mode")
	}
	name := os.Getenv("STIFACE_BUCKET")
	if name == "" {
		t.Skip("missing STIFACE_BUCKET environment variable")
	}
	ctx := context.Background()
	c, err := storage.NewClient(ctx)
	if err != nil {
		t.Fatal(err)
	}
	client := AdaptClient(c)
	defer client.Close()
	bkt := client.Bucket(name)
	basicTests(t, name, bkt)
}

func basicTests(t *testing.T, bucketName string, bkt BucketHandle) {
	ctx := context.Background()
	attrs, err := bkt.Attrs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := attrs.Name, bucketName; got != want {
		t.Errorf("name: got %v, want %v", got, want)
	}

	const contents = "hello, stiface"
	obj := bkt.Object("stiface-test")
	w := obj.NewWriter(ctx)
	if _, err := fmt.Fprint(w, contents); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	bytes := readObject(t, obj)
	if got, want := string(bytes), contents; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if err := obj.Delete(ctx); err != nil {
		t.Errorf("deleting: %v", err)
	}
}

func readObject(t *testing.T, obj ObjectHandle) []byte {
	r, err := obj.NewReader(context.Background())
	if err != nil {
		t.Fatalf("reading %v: %v", obj, err)
	}
	defer r.Close()
	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatalf("reading %v: %v", obj, err)
	}
	return bytes
}

// This test demonstrates how to use this package to create a simple fake for the storage client.
func TestFake(t *testing.T) {
	ctx := context.Background()
	client := newFakeClient()

	bkt := client.Bucket("my-bucket")
	if err := bkt.Create(ctx, "my-project", nil); err != nil {
		t.Fatal(err)
	}
	basicTests(t, "my-bucket", bkt)
}

type fakeClient struct {
	Client
	buckets map[string]*fakeBucket
}

type fakeBucket struct {
	attrs   *storage.BucketAttrs
	objects map[string][]byte
}

func newFakeClient() Client {
	return &fakeClient{buckets: map[string]*fakeBucket{}}
}

func (c *fakeClient) Bucket(name string) BucketHandle {
	return fakeBucketHandle{c: c, name: name}
}

type fakeBucketHandle struct {
	BucketHandle
	c    *fakeClient
	name string
}

func (b fakeBucketHandle) Create(_ context.Context, _ string, attrs *storage.BucketAttrs) error {
	if _, ok := b.c.buckets[b.name]; ok {
		return fmt.Errorf("bucket %q already exists", b.name)
	}
	if attrs == nil {
		attrs = &storage.BucketAttrs{}
	}
	attrs.Name = b.name
	b.c.buckets[b.name] = &fakeBucket{attrs: attrs, objects: map[string][]byte{}}
	return nil
}

func (b fakeBucketHandle) Attrs(context.Context) (*storage.BucketAttrs, error) {
	bkt, ok := b.c.buckets[b.name]
	if !ok {
		return nil, fmt.Errorf("bucket %q does not exist", b.name)
	}
	return bkt.attrs, nil
}

func (b fakeBucketHandle) Object(name string) ObjectHandle {
	return fakeObjectHandle{c: b.c, bucketName: b.name, name: name}
}

type fakeObjectHandle struct {
	ObjectHandle
	c          *fakeClient
	bucketName string
	name       string
}

func (o fakeObjectHandle) NewReader(context.Context) (Reader, error) {
	bkt, ok := o.c.buckets[o.bucketName]
	if !ok {
		return nil, fmt.Errorf("bucket %q not found", o.bucketName)
	}
	contents, ok := bkt.objects[o.name]
	if !ok {
		return nil, fmt.Errorf("object %q not found in bucket %q", o.name, o.bucketName)
	}
	return fakeReader{r: bytes.NewReader(contents)}, nil
}

func (o fakeObjectHandle) Delete(context.Context) error {
	bkt, ok := o.c.buckets[o.bucketName]
	if !ok {
		return fmt.Errorf("bucket %q not found", o.bucketName)
	}
	delete(bkt.objects, o.name)
	return nil
}

type fakeReader struct {
	Reader
	r *bytes.Reader
}

func (r fakeReader) Read(buf []byte) (int, error) {
	return r.r.Read(buf)
}

func (r fakeReader) Close() error {
	return nil
}

func (o fakeObjectHandle) NewWriter(context.Context) Writer {
	return &fakeWriter{obj: o}
}

type fakeWriter struct {
	Writer
	obj fakeObjectHandle
	buf bytes.Buffer
}

func (w *fakeWriter) Write(data []byte) (int, error) {
	return w.buf.Write(data)
}

func (w *fakeWriter) Close() error {
	bkt, ok := w.obj.c.buckets[w.obj.bucketName]
	if !ok {
		return fmt.Errorf("bucket %q not found", w.obj.bucketName)
	}
	bkt.objects[w.obj.name] = w.buf.Bytes()
	return nil
}
