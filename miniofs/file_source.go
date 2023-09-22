package minio

import (
	"bytes"
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"io"
	"net/http"
	"os"
)

type minioFileResource struct {
	ctx context.Context

	fs *Fs

	bucket   *minio.BucketInfo
	name     string
	fileMode os.FileMode

	currentGcsSize int64
	offset         int64
	reader         io.ReadCloser
	writer         io.WriteCloser

	closed bool
}

func (o *minioFileResource) Close() error {
	o.closed = true
	// TODO rawGcsObjectsMap ?
	return o.maybeCloseIo()
}

func (o *minioFileResource) maybeCloseIo() error {
	if err := o.maybeCloseReader(); err != nil {
		return fmt.Errorf("error closing reader: %v", err)
	}
	if err := o.maybeCloseWriter(); err != nil {
		return fmt.Errorf("error closing writer: %v", err)
	}

	return nil
}

func (o *minioFileResource) maybeCloseReader() error {
	if o.reader == nil {
		return nil
	}
	if err := o.reader.Close(); err != nil {
		return err
	}
	o.reader = nil
	return nil
}

func (o *minioFileResource) maybeCloseWriter() error {
	if o.writer == nil {
		return nil
	}

	// In cases of partial writes (e.g. to the middle of a file stream), we need to
	// append any remaining data from the original file before we close the reader (and
	// commit the results.)
	// For small writes it can be more efficient
	// to keep the original reader but that is for another iteration
	if o.currentGcsSize > o.offset {

	}

	if err := o.writer.Close(); err != nil {
		return err
	}
	o.writer = nil
	return nil
}

func (o *minioFileResource) ReadAt(p []byte, off int64) (n int, err error) {
	if cap(p) == 0 {
		return 0, nil
	}

	// Assume that if the reader is open; it is at the correct offset
	// a good performance assumption that we must ensure holds
	if off == o.offset && o.reader != nil {
		n, err = o.reader.Read(p)
		o.offset += int64(n)
		return n, err
	}

	// If any writers have written anything; commit it first so we can read it back.
	if err = o.maybeCloseIo(); err != nil {
		return 0, err
	}

	opts := minio.GetObjectOptions{
		PartNumber: int(off),
	}
	r, err := o.fs.client.GetObject(o.ctx, o.bucket.Name, o.name, opts)
	if err != nil {
		return 0, err
	}
	//defer r.Close()

	o.reader = r
	o.offset = off

	read, err := o.reader.Read(p)
	o.offset += int64(read)
	return read, err
}

func (o *minioFileResource) WriteAt(b []byte, off int64) (n int, err error) {
	// If the writer is opened and at the correct offset we're good!
	if off == o.offset && o.writer != nil {
		n, err = o.writer.Write(b)
		o.offset += int64(n)
		return n, err
	}

	// Ensure readers must be re-opened and that if a writer is active at another
	// offset it is first committed before we do a "seek" below
	if err = o.maybeCloseIo(); err != nil {
		return 0, err
	}

	// WriteAt to a non existing file
	if off > 0 {
		return 0, ErrOutOfRange
	}
	o.offset = off
	//o.writer =

	// byt 写入 buffer
	buffer := bytes.NewReader(b)

	// 写入 minio
	opts := minio.PutObjectOptions{
		ContentType: http.DetectContentType(b),
	}
	_, err = o.fs.client.PutObject(o.ctx, o.bucket.Name, o.name, buffer, buffer.Size(), opts)
	if err != nil {
		return 0, err
	}

	o.offset += int64(buffer.Len())
	return buffer.Len(), nil
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func (o *minioFileResource) Truncate(wantedSize int64) error {
	if wantedSize < 0 {
		return ErrOutOfRange
	}

	if err := o.maybeCloseIo(); err != nil {
		return err
	}

	r, err := o.fs.client.GetObject(o.ctx, o.bucket.Name, o.name, minio.GetObjectOptions{})
	if err != nil {
		return err
	}
	defer r.Close()

	//if r.Size() < wantedSize {
	//	return ErrOutOfRange
	//}

	srcBuffer := make([]byte, wantedSize)
	_, err = o.ReadAt(srcBuffer, wantedSize)
	if err != nil {
		return err
	}
	src := bytes.NewReader(srcBuffer)

	_, err = o.fs.client.PutObject(o.ctx, o.bucket.Name, o.name, src, src.Size(), minio.PutObjectOptions{})

	return err
}
