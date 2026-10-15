package gcsfs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"cloud.google.com/go/storage"

	"github.com/spf13/afero/gcsfs/internal/stiface"
)

// spyWriter records lifecycle calls so tests can assert that pending uploads
// are aborted (CloseWithError) instead of being leaked.
type spyWriter struct {
	stiface.Writer

	buf         bytes.Buffer
	closeCount  int
	closeErrors []error
	writeErr    error
}

func (w *spyWriter) Write(p []byte) (int, error) {
	if w.writeErr != nil {
		return 0, w.writeErr
	}
	return w.buf.Write(p)
}

func (w *spyWriter) Close() error {
	w.closeCount++
	return nil
}

func (w *spyWriter) CloseWithError(err error) error {
	w.closeCount++
	w.closeErrors = append(w.closeErrors, err)
	return nil
}

func (w *spyWriter) ObjectAttrs() *storage.ObjectAttrs { return &storage.ObjectAttrs{} }
func (w *spyWriter) SetChunkSize(int)                  {}
func (w *spyWriter) SetProgressFunc(func(int64))       {}
func (w *spyWriter) SetCRC32C(uint32)                  {}
func (w *spyWriter) Attrs() *storage.ObjectAttrs        { return &storage.ObjectAttrs{} }

// spyReader records Close calls so tests can assert readers are released.
type spyReader struct {
	stiface.Reader

	data       []byte
	pos        int
	remain     int64
	closeCount int
	readErr    error
}

func (r *spyReader) Read(p []byte) (int, error) {
	if r.readErr != nil {
		return 0, r.readErr
	}
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *spyReader) Close() error            { r.closeCount++; return nil }
func (r *spyReader) Size() int64             { return int64(len(r.data)) }
func (r *spyReader) Remain() int64           { return r.remain }
func (r *spyReader) ContentType() string     { return "" }
func (r *spyReader) ContentEncoding() string { return "" }
func (r *spyReader) CacheControl() string    { return "" }

// spyObject is a minimal ObjectHandle that hands out the spies above.
type spyObject struct {
	stiface.ObjectHandle

	attrs         *storage.ObjectAttrs
	attrsErr      error
	readerErr     error
	readerReadErr error
	rangeErr      error
	rangeReadErr  error
	rangeRemain   int64
	rangeData     []byte

	writers      []*spyWriter
	readers      []*spyReader
	rangeReaders []*spyReader
}

func (o *spyObject) Attrs(context.Context) (*storage.ObjectAttrs, error) {
	if o.attrsErr != nil {
		return nil, o.attrsErr
	}
	if o.attrs != nil {
		return o.attrs, nil
	}
	return &storage.ObjectAttrs{}, nil
}

func (o *spyObject) NewWriter(context.Context) stiface.Writer {
	w := &spyWriter{}
	o.writers = append(o.writers, w)
	return w
}

func (o *spyObject) NewReader(context.Context) (stiface.Reader, error) {
	if o.readerErr != nil {
		return nil, o.readerErr
	}
	r := &spyReader{remain: o.rangeRemain, readErr: o.readerReadErr}
	o.readers = append(o.readers, r)
	return r, nil
}

func (o *spyObject) NewRangeReader(context.Context, int64, int64) (stiface.Reader, error) {
	if o.rangeErr != nil {
		return nil, o.rangeErr
	}
	r := &spyReader{data: o.rangeData, remain: o.rangeRemain, readErr: o.rangeReadErr}
	o.rangeReaders = append(o.rangeReaders, r)
	return r, nil
}

func newSpyResource(obj *spyObject, gcsSize, offset int64) *gcsFileResource {
	return &gcsFileResource{
		ctx:            context.Background(),
		fs:             &Fs{},
		obj:            obj,
		name:           "object",
		fileMode:       0o644,
		currentGcsSize: gcsSize,
		offset:         offset,
	}
}

func expectClosed(t *testing.T, count int, what string) {
	t.Helper()
	if count != 1 {
		t.Fatalf("%s: expected exactly 1 close, got %d", what, count)
	}
}

// WriteAt used to return before assigning o.writer, leaking the pending upload.
func TestWriteAtAbortsWriterWhenAttrsFails(t *testing.T) {
	obj := &spyObject{attrsErr: errors.New("transient storage error")}
	res := newSpyResource(obj, 0, 0)

	_, err := res.WriteAt([]byte("x"), 5)
	if err == nil {
		t.Fatal("expected an error")
	}
	if len(obj.writers) != 1 {
		t.Fatalf("expected 1 writer, got %d", len(obj.writers))
	}
	expectClosed(t, obj.writers[0].closeCount, "pending writer")
	if len(obj.writers[0].closeErrors) == 0 {
		t.Fatal("writer must be aborted with CloseWithError")
	}
}

// A write beyond the current object size must not leak the pending writer.
func TestWriteAtAbortsWriterWhenOutOfRange(t *testing.T) {
	obj := &spyObject{attrs: &storage.ObjectAttrs{Size: 2}}
	res := newSpyResource(obj, 0, 0)

	if _, err := res.WriteAt([]byte("x"), 5); !errors.Is(err, ErrOutOfRange) {
		t.Fatalf("expected ErrOutOfRange, got %v", err)
	}
	if len(obj.writers) != 1 {
		t.Fatalf("expected 1 writer, got %d", len(obj.writers))
	}
	expectClosed(t, obj.writers[0].closeCount, "pending writer")
}

// A failed prefix copy must release both the source reader and the writer.
func TestWriteAtReleasesReaderAndWriterWhenCopyFails(t *testing.T) {
	obj := &spyObject{
		attrs:         &storage.ObjectAttrs{Size: 100},
		readerReadErr: errors.New("read failed"),
	}
	res := newSpyResource(obj, 0, 0)

	if _, err := res.WriteAt([]byte("x"), 10); err == nil {
		t.Fatal("expected an error")
	}
	if len(obj.readers) != 1 {
		t.Fatalf("expected 1 reader, got %d", len(obj.readers))
	}
	expectClosed(t, obj.readers[0].closeCount, "prefix reader")
	if len(obj.writers) != 1 {
		t.Fatalf("expected 1 writer, got %d", len(obj.writers))
	}
	expectClosed(t, obj.writers[0].closeCount, "pending writer")
}

// A failed copy in Truncate must release the range reader and abort the writer.
func TestTruncateReleasesReaderAndWriterWhenCopyFails(t *testing.T) {
	obj := &spyObject{
		rangeData:    []byte("hello"),
		rangeRemain:  5,
		rangeReadErr: errors.New("read failed"),
	}
	res := newSpyResource(obj, 5, 5)

	if err := res.Truncate(10); err == nil {
		t.Fatal("expected an error")
	}
	if len(obj.rangeReaders) != 1 {
		t.Fatalf("expected 1 range reader, got %d", len(obj.rangeReaders))
	}
	expectClosed(t, obj.rangeReaders[0].closeCount, "range reader")
	if len(obj.writers) != 1 {
		t.Fatalf("expected 1 writer, got %d", len(obj.writers))
	}
	expectClosed(t, obj.writers[0].closeCount, "pending writer")
	if len(obj.writers[0].closeErrors) == 0 {
		t.Fatal("writer must be aborted with CloseWithError")
	}
}

// A successful Truncate closes both handles exactly once.
func TestTruncateClosesReaderAndWriterOnSuccess(t *testing.T) {
	obj := &spyObject{
		rangeData:   []byte("hello"),
		rangeRemain: 5,
	}
	res := newSpyResource(obj, 5, 5)

	if err := res.Truncate(10); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(obj.rangeReaders) != 1 {
		t.Fatalf("expected 1 range reader, got %d", len(obj.rangeReaders))
	}
	expectClosed(t, obj.rangeReaders[0].closeCount, "range reader")
	if len(obj.writers) != 1 {
		t.Fatalf("expected 1 writer, got %d", len(obj.writers))
	}
	expectClosed(t, obj.writers[0].closeCount, "writer")
	if len(obj.writers[0].closeErrors) != 0 {
		t.Fatal("successful close must not use CloseWithError")
	}
}

// maybeCloseWriter must release the range reader even when no data is copied.
func TestMaybeCloseWriterClosesCurrentFileWithoutRemainder(t *testing.T) {
	obj := &spyObject{rangeRemain: 0}
	res := newSpyResource(obj, 10, 0)
	writer := &spyWriter{}
	res.writer = writer

	if err := res.maybeCloseWriter(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(obj.rangeReaders) != 1 {
		t.Fatalf("expected 1 range reader, got %d", len(obj.rangeReaders))
	}
	expectClosed(t, obj.rangeReaders[0].closeCount, "range reader")
	expectClosed(t, writer.closeCount, "writer")
}

// maybeCloseWriter must release the range reader after copying the remainder.
func TestMaybeCloseWriterClosesCurrentFileAfterCopy(t *testing.T) {
	obj := &spyObject{rangeData: []byte("tail"), rangeRemain: 4}
	res := newSpyResource(obj, 10, 0)
	writer := &spyWriter{}
	res.writer = writer

	if err := res.maybeCloseWriter(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(obj.rangeReaders) != 1 {
		t.Fatalf("expected 1 range reader, got %d", len(obj.rangeReaders))
	}
	expectClosed(t, obj.rangeReaders[0].closeCount, "range reader")
	expectClosed(t, writer.closeCount, "writer")
}
