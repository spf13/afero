// Copyright © 2021 Vasily Ovchinnikov <vasily@remerge.io>.
//
// A set of stiface-based mocks, replicating the GCS behavior, to make the tests not require any
// internet connection or real buckets.
// It is **not** a comprehensive set of mocks to test anything and everything GCS-related, rather
// a very tailored one for the current implementation - thus the tests, written with the use of
// these mocks are more of regression ones.
// If any GCS behavior changes and breaks the implementation, then it should first be adjusted by
// switching over to a real bucket - and then the mocks have to be adjusted to match the
// implementation.

package afero

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/spf13/afero/gcsfs"

	"cloud.google.com/go/storage"
	"github.com/googleapis/google-cloud-go-testing/storage/stiface"
	"google.golang.org/api/iterator"
)

// sets filesystem separators to the one, expected (and hard-coded) in the tests
func normSeparators(s string) string {
	return strings.Replace(s, "\\", "/", -1)
}

type clientMock struct {
	stiface.Client
	fs Fs
}

func newClientMock() *clientMock {
	return &clientMock{fs: NewMemMapFs()}
}

func (m *clientMock) Bucket(name string) stiface.BucketHandle {
	return &bucketMock{bucketName: name, fs: m.fs}
}

type bucketMock struct {
	stiface.BucketHandle

	bucketName string

	fs Fs
}

func (m *bucketMock) Attrs(context.Context) (*storage.BucketAttrs, error) {
	return &storage.BucketAttrs{}, nil
}

func (m *bucketMock) Object(name string) stiface.ObjectHandle {
	return &objectMock{name: name, fs: m.fs}
}

func (m *bucketMock) Objects(_ context.Context, q *storage.Query) (it stiface.ObjectIterator) {
	return &objectItMock{name: q.Prefix, fs: m.fs}
}

type objectMock struct {
	stiface.ObjectHandle

	name string
	fs   Fs
}

func (o *objectMock) NewWriter(_ context.Context) stiface.Writer {
	return &writerMock{name: o.name, fs: o.fs}
}

func (o *objectMock) NewRangeReader(_ context.Context, offset, length int64) (stiface.Reader, error) {
	if o.name == "" {
		return nil, gcsfs.ErrEmptyObjectName
	}

	file, err := o.fs.Open(o.name)
	if err != nil {
		return nil, err
	}

	if offset > 0 {
		_, err = file.Seek(offset, io.SeekStart)
		if err != nil {
			return nil, err
		}
	}

	res := &readerMock{file: file}
	if length > -1 {
		res.buf = make([]byte, length)
		_, err = file.Read(res.buf)
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (o *objectMock) Delete(_ context.Context) error {
	if o.name == "" {
		return gcsfs.ErrEmptyObjectName
	}
	return o.fs.Remove(o.name)
}

func (o *objectMock) Attrs(_ context.Context) (*storage.ObjectAttrs, error) {
	if o.name == "" {
		return nil, gcsfs.ErrEmptyObjectName
	}

	info, err := o.fs.Stat(o.name)
	if err != nil {
		pathError, ok := err.(*os.PathError)
		if ok {
			if pathError.Err == os.ErrNotExist {
				return nil, storage.ErrObjectNotExist
			}
		}

		return nil, err
	}

	res := &storage.ObjectAttrs{Name: normSeparators(o.name), Size: info.Size(), Updated: info.ModTime()}

	if info.IsDir() {
		// we have to mock it here, because of FileInfo logic
		return nil, gcsfs.ErrObjectDoesNotExist
	}

	return res, nil
}

type writerMock struct {
	stiface.Writer

	name string
	fs   Fs

	file File
}

func (w *writerMock) Write(p []byte) (n int, err error) {
	if w.file == nil {
		w.file, err = w.fs.Create(w.name)
		if err != nil {
			return 0, err
		}
	}

	return w.file.Write(p)
}

func (w *writerMock) Close() error {
	if w.file == nil {
		var err error
		if strings.HasSuffix(w.name, "/") {
			err = w.fs.Mkdir(w.name, 0755)
			if err != nil {
				return err
			}
		} else {
			_, err = w.Write([]byte{})
			if err != nil {
				return err
			}
		}
	}
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

type readerMock struct {
	stiface.Reader

	file File

	buf []byte
}

func (r *readerMock) Remain() int64 {
	return 0
}

func (r *readerMock) Read(p []byte) (int, error) {
	if r.buf != nil {
		copy(p, r.buf)
		return len(r.buf), nil
	}
	return r.file.Read(p)
}

func (r *readerMock) Close() error {
	return r.file.Close()
}

type objectItMock struct {
	stiface.ObjectIterator

	name string
	fs   Fs

	dir   File
	infos []*storage.ObjectAttrs
}

func (it *objectItMock) Next() (*storage.ObjectAttrs, error) {
	var err error
	if it.dir == nil {
		it.dir, err = it.fs.Open(it.name)
		if err != nil {
			return nil, err
		}

		var isDir bool
		isDir, err = IsDir(it.fs, it.name)
		if err != nil {
			return nil, err
		}

		it.infos = []*storage.ObjectAttrs{}

		if !isDir {
			var info os.FileInfo
			info, err = it.dir.Stat()
			if err != nil {
				return nil, err
			}
			it.infos = append(it.infos, &storage.ObjectAttrs{Name: normSeparators(info.Name()), Size: info.Size(), Updated: info.ModTime()})
		} else {
			var fInfos []os.FileInfo
			fInfos, err = it.dir.Readdir(0)
			if err != nil {
				return nil, err
			}
			if it.name != "" {
				it.infos = append(it.infos, &storage.ObjectAttrs{
					Prefix: normSeparators(it.name) + "/",
				})
			}

			for _, info := range fInfos {
				it.infos = append(it.infos, &storage.ObjectAttrs{Name: normSeparators(info.Name()), Size: info.Size(), Updated: info.ModTime()})
			}
		}
	}

	if len(it.infos) == 0 {
		return nil, iterator.Done
	}

	res := it.infos[0]
	it.infos = it.infos[1:]

	return res, err
}
