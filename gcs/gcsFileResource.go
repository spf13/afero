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

package gcs

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
)

// gcsFileResource represents a singleton version of each GCS object;
// Google cloud storage allows users to open multiple writers(!) to the same
// underlying resource, once the write is closed the written stream is commented. We are doing
// some magic where we reand and write to the same file which requires syncronization
// of the underlying resource.
type gcsFileResource struct {
	ctx context.Context
	fs  *GcsFs

	obj  *storage.ObjectHandle
	name string

	currentGcsSize int64
	offset         int64
	reader         io.ReadCloser
	writer         io.WriteCloser

	closed bool
}

func (o *gcsFileResource) Close() error {
	o.closed = true
	delete(o.fs.rawGcsObjects, o.name)
	return o.maybeCloseIo()
}

func (o *gcsFileResource) maybeCloseIo() error {
	o.maybeCloseReader()
	return o.maybeCloseWriter()
}

func (o *gcsFileResource) maybeCloseReader() {
	if o.reader == nil {
		return
	}
	o.reader.Close()
	o.reader = nil
}

func (o *gcsFileResource) maybeCloseWriter() error {
	if o.writer == nil {
		return nil
	}

	// In cases of partial writes (e.g. to the middle of a file stream), we need to
	// append any remaining data from the orignial file before we close the reader (and
	// commit the results.)
	// For small writes it can be more efficient
	// to keep the original reader but that is for another iteration
	if o.currentGcsSize > o.offset {
		currentFile, err := o.obj.NewRangeReader(o.ctx, o.offset, -1)
		if err != nil {
			return fmt.Errorf(
				"Couldn't simulate a partial write; the closing (and thus"+
					" the whole file write) is NOT commited to GCS. %v", err)
		}
		if err == nil && currentFile != nil && currentFile.Remain() > 0 {
			io.Copy(o.writer, currentFile)
		}
	}

	o.writer.Close()
	o.writer = nil
	return nil
}

func (o *gcsFileResource) ReadAt(p []byte, off int64) (n int, err error) {
	if cap(p) == 0 {
		return 0, nil
	}

	// Assume that if the reader is open; it is at the correct fhoffset
	// a good performance assumption that we must ensure holds
	if off == o.offset && o.reader != nil {
		read, err := o.reader.Read(p)
		o.offset += int64(read)
		return read, err
	}

	//If any writers have written anything; commit it first so we can read it back.
	o.maybeCloseIo()

	//Then read at the correct offset.
	r, err := o.obj.NewRangeReader(o.ctx, off, -1)
	if err != nil {
		return 0, err
	}
	o.reader = r
	o.offset = off

	read, err := o.reader.Read(p)
	o.offset += int64(read)
	return read, err
}

func (o *gcsFileResource) WriteAt(b []byte, off int64) (n int, err error) {
	//If the writer is opened and at the correct offset we're good!
	if off == o.offset && o.writer != nil {
		written, err := o.writer.Write(b)
		o.offset += int64(written)
		return written, err
	}

	// Ensure readers must be re-opened and that if a writer is active at another
	// offset it is first commited before we do a "seek" below
	o.maybeCloseIo()

	w := o.obj.NewWriter(o.ctx)
	// TRIGGER WARNING: This can seem like a hack but it works thanks
	// to GCS strong consistency. We will open and write to the same file; First when the
	// writer is closed will the content get commented to GCS.
	// The general idea is this:
	// Objectv1[:offset] -> Objectv2
	// newData1 -> Objectv2
	// Objectv1[offset+len(newData1):] -> Objectv2
	// Objectv2.Close
	//
	// It will however require a download and upload of the original file but it
	// can't be avoided if we should support seek-write-operations on GCS.
	objAttrs, err := o.obj.Attrs(o.ctx)
	if err != nil {
		if off > 0 {
			return 0, err // WriteAt to a non existing file
		}
		o.currentGcsSize = 0
	} else {
		o.currentGcsSize = int64(objAttrs.Size)
	}

	if off > o.currentGcsSize {
		return 0, ErrOutOfRange
	}

	if off > 0 {
		r, err := o.obj.NewReader(o.ctx)
		if err != nil {
			return 0, err
		}
		if _, err := io.CopyN(w, r, off); err != nil {
			return 0, err
		}
		r.Close()
	}

	o.writer = w
	o.offset = off

	written, err := o.writer.Write(b)
	o.offset += int64(written)
	return written, err
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func (o *gcsFileResource) Truncate(wantedSize int64) error {
	if wantedSize < 0 {
		return ErrOutOfRange
	}

	o.maybeCloseIo()

	r, err := o.obj.NewRangeReader(o.ctx, 0, wantedSize)
	if err != nil {
		return err
	}

	w := o.obj.NewWriter(o.ctx)
	written, err := io.Copy(w, r)
	if err != nil {
		return err
	}

	MAX_WRITE_SIZE := 10000
	for written < wantedSize {
		//Bulk up padding writes
		paddingBytes := bytes.Repeat([]byte(" "), min(MAX_WRITE_SIZE, int(wantedSize-written)))
		if w, err := w.Write(paddingBytes); err != nil {
			return err
		} else {
			written += int64(w)
		}
	}
	r.Close()
	return w.Close()
}
