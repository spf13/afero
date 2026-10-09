package zipfs

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"runtime"
	"sync"
	"testing"
)

func openStreamFixture(t *testing.T, data []byte) *File {
	t.Helper()
	var archive bytes.Buffer
	w := zip.NewWriter(&archive)
	entry, err := w.Create("data")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := zip.NewReader(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	if err != nil {
		t.Fatal(err)
	}
	f, err := New(r).Open("data")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Error(err)
		}
	})
	return f.(*File)
}

func TestLargeSequentialReadUsesBoundedMemory(t *testing.T) {
	const size = 16 << 20
	f := openStreamFixture(t, bytes.Repeat([]byte("x"), size))
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	n, err := io.Copy(io.Discard, f)
	runtime.ReadMemStats(&after)
	if err != nil || n != size {
		t.Fatalf("read %d bytes: %v", n, err)
	}
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > size/2 {
		t.Fatalf("sequential read allocated %d bytes for a %d-byte entry", allocated, size)
	}
}

func TestStreamSeekAndReadAt(t *testing.T) {
	f := openStreamFixture(t, []byte("abcdefghij"))
	p := make([]byte, 3)
	if n, err := f.Read(p); n != 3 || err != nil || string(p) != "abc" {
		t.Fatalf("initial read: %d %q %v", n, p, err)
	}
	if n, err := f.ReadAt(p, 7); n != 3 || err != nil || string(p) != "hij" {
		t.Fatalf("ReadAt: %d %q %v", n, p, err)
	}
	if n, err := f.Read(p); n != 3 || err != nil || string(p) != "def" {
		t.Fatalf("cursor moved: %d %q %v", n, p, err)
	}
	if _, err := f.Seek(1, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	if n, err := f.Read(p); n != 3 || err != nil || string(p) != "bcd" {
		t.Fatalf("backward seek: %d %q %v", n, p, err)
	}
	if _, err := f.Seek(-2, io.SeekEnd); err != nil {
		t.Fatal(err)
	}
	if n, err := f.Read(p); n != 2 || !errors.Is(err, io.EOF) || string(p[:n]) != "ij" {
		t.Fatalf("tail read: %d %q %v", n, p, err)
	}
	if n, err := f.ReadAt(p, 100); n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("past end: %d %v", n, err)
	}
	if _, err := f.ReadAt(p, -1); err == nil {
		t.Fatal("negative offset accepted")
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	if n, err := f.Read(p); n != 3 || err != nil || string(p) != "abc" {
		t.Fatalf("restart: %d %q %v", n, p, err)
	}
}

func TestConcurrentReadAtPreservesCursor(t *testing.T) {
	f := openStreamFixture(t, []byte("abcdefghij"))
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Go(func() {
			p := make([]byte, 2)
			n, err := f.ReadAt(p, int64(i))
			if n != 2 || err != nil || string(p) != "abcdefghij"[i:i+2] {
				t.Errorf("ReadAt(%d): %d %q %v", i, n, p, err)
			}
		})
	}
	wg.Wait()
	if pos, err := f.Seek(0, io.SeekCurrent); pos != 0 || err != nil {
		t.Fatalf("cursor: %d %v", pos, err)
	}
}

func TestStreamPreservesChecksumFailure(t *testing.T) {
	f := openStreamFixture(t, []byte("checksum"))
	f.zipfile.CRC32++
	p := make([]byte, 16)
	if n, err := f.Read(p); n != 8 || !errors.Is(err, zip.ErrChecksum) {
		t.Fatalf("Read: %d bytes, %v", n, err)
	}
	if n, err := f.ReadAt(p, 0); n != 8 || !errors.Is(err, zip.ErrChecksum) {
		t.Fatalf("ReadAt: %d bytes, %v", n, err)
	}
}
