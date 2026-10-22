package zipfs

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

func TestFileRead(t *testing.T) {
	zrc, err := zip.OpenReader("testdata/small.zip")
	if err != nil {
		t.Fatal(err)
	}
	zfs := New(&zrc.Reader)
	f, err := zfs.Open("smallFile")
	if err != nil {
		t.Fatal(err)
	}
	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	chunkSize := info.Size() * 2 // read with extra large buffer

	buf := make([]byte, chunkSize)
	n, err := f.Read(buf)
	if err != io.EOF {
		t.Fatal("Failed to read file to completion:", err)
	}
	if n != int(info.Size()) {
		t.Errorf("Expected read length to be %d, found: %d", info.Size(), n)
	}

	// read a second time to check f.offset and f.buf are correct
	buf = make([]byte, chunkSize)
	n, err = f.Read(buf)
	if err != io.EOF {
		t.Fatal("Failed to read a fully read file:", err)
	}
	if n != 0 {
		t.Errorf("Expected read length to be 0, found: %d", n)
	}
}

func TestFileReadAt_BeyondSize(t *testing.T) {
	zrc, err := zip.OpenReader("testdata/small.zip")
	if err != nil {
		t.Fatal(err)
	}
	defer zrc.Close()
	zfs := New(&zrc.Reader)
	f, err := zfs.Open("smallFile")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	buf := make([]byte, 10)
	n, err := f.ReadAt(buf, 100)
	if n != 0 || err != io.EOF {
		t.Errorf("ReadAt beyond file size: got n=%d, err=%v; want n=0, err=io.EOF", n, err)
	}
}

func TestFileReadAt_NegativeOffset(t *testing.T) {
	zrc, err := zip.OpenReader("testdata/small.zip")
	if err != nil {
		t.Fatal(err)
	}
	defer zrc.Close()
	zfs := New(&zrc.Reader)
	f, err := zfs.Open("smallFile")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	buf := make([]byte, 10)
	n, err := f.ReadAt(buf, -1)
	if err == nil {
		t.Errorf("ReadAt with negative offset: got n=%d, err=nil; want error", n)
	}
}

func TestFile_PatchedUncompressedSize(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.CreateHeader(&zip.FileHeader{
		Name:   "patched.txt",
		Method: zip.Store,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("hello world")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	raw := buf.Bytes()
	// Patch uncompressed size to 1000 in both Local File Header (offset 22)
	// and Central Directory File Header (offset 24 after PK\x01\x02).
	locSig := []byte{0x50, 0x4b, 0x03, 0x04}
	cdSig := []byte{0x50, 0x4b, 0x01, 0x02}
	locIdx := bytes.Index(raw, locSig)
	if locIdx == -1 {
		t.Fatal("local file header not found")
	}
	binary.LittleEndian.PutUint32(raw[locIdx+22:], 1000)

	cdIdx := bytes.Index(raw, cdSig)
	if cdIdx == -1 {
		t.Fatal("central directory header not found")
	}
	binary.LittleEndian.PutUint32(raw[cdIdx+24:], 1000)

	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	zfs := New(zr)
	f, err := zfs.Open("patched.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	// Seek to 500 then Read -> should return 0, io.EOF and no panic
	seekPos, err := f.Seek(500, io.SeekStart)
	if err != nil {
		t.Fatalf("Seek failed: %v", err)
	}
	if seekPos != 500 {
		t.Fatalf("Seek position = %d, want 500", seekPos)
	}

	readBuf := make([]byte, 100)
	n, err := f.Read(readBuf)
	if n != 0 || err != io.EOF {
		t.Errorf("Read after seek past stored data: got n=%d, err=%v; want n=0, err=io.EOF", n, err)
	}

	// ReadAt with off=500 -> should return 0, io.EOF and no panic
	n, err = f.ReadAt(readBuf, 500)
	if n != 0 || err != io.EOF {
		t.Errorf("ReadAt with off=500: got n=%d, err=%v; want n=0, err=io.EOF", n, err)
	}
}
