package afero

import (
	"bytes"
	"io"
	"testing"
)

func TestMemMapFsFileImplementsReaderFrom(t *testing.T) {
	fs := NewMemMapFs()
	dst, err := fs.Create("dst.txt")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer dst.Close()

	rf, ok := dst.(io.ReaderFrom)
	if !ok {
		t.Fatal("MemMapFs file should implement io.ReaderFrom")
	}

	n, err := rf.ReadFrom(bytes.NewReader([]byte("payload")))
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if n != int64(len("payload")) {
		t.Fatalf("ReadFrom bytes: got %d", n)
	}

	got, err := ReadFile(fs, "dst.txt")
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(got) != "payload" {
		t.Fatalf("file contents: got %q", string(got))
	}
}

func TestMemMapFsIoCopyUsesReadFrom(t *testing.T) {
	fs := NewMemMapFs()
	dst, err := fs.Create("dst.txt")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer dst.Close()

	n, err := io.Copy(dst, bytes.NewReader([]byte("payload")))
	if err != nil {
		t.Fatalf("copy: %v", err)
	}
	if n != int64(len("payload")) {
		t.Fatalf("copied bytes: got %d", n)
	}

	got, err := ReadFile(fs, "dst.txt")
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(got) != "payload" {
		t.Fatalf("file contents: got %q", string(got))
	}
}
