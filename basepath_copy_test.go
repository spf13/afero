package afero

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type optionalIfaceFs struct {
	Fs
	openWrap     func(File) File
	createWrap   func(File) File
	openFileWrap func(File) File
}

func (f optionalIfaceFs) Open(name string) (File, error) {
	file, err := f.Fs.Open(name)
	if err != nil {
		return nil, err
	}
	if f.openWrap != nil {
		return f.openWrap(file), nil
	}
	return file, nil
}

func (f optionalIfaceFs) Create(name string) (File, error) {
	file, err := f.Fs.Create(name)
	if err != nil {
		return nil, err
	}
	if f.createWrap != nil {
		return f.createWrap(file), nil
	}
	return file, nil
}

func (f optionalIfaceFs) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	file, err := f.Fs.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}
	if f.openFileWrap != nil {
		return f.openFileWrap(file), nil
	}
	return file, nil
}

type writerToTrackingFile struct {
	File
	called *bool
}

func (f *writerToTrackingFile) WriteTo(w io.Writer) (int64, error) {
	*f.called = true
	return io.Copy(w, f.File)
}

type readerFromTrackingFile struct {
	File
	called *bool
}

func (f *readerFromTrackingFile) ReadFrom(r io.Reader) (int64, error) {
	*f.called = true
	return io.Copy(f.File, r)
}

type slowReader struct {
	data []byte
}

func (r *slowReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

func TestBasePathFileForwardsWriterTo(t *testing.T) {
	base := NewMemMapFs()
	if err := base.MkdirAll("/root", 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := WriteFile(base, "/root/src.txt", []byte("payload"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	var called bool
	fs := optionalIfaceFs{
		Fs: base,
		openWrap: func(file File) File {
			return &writerToTrackingFile{File: file, called: &called}
		},
	}
	bp := NewBasePathFs(fs, "/root")

	src, err := bp.Open("src.txt")
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	defer src.Close()

	if _, ok := src.(io.WriterTo); !ok {
		t.Fatal("BasePathFile should expose io.WriterTo when the wrapped file implements it")
	}

	var dst bytes.Buffer
	n, err := io.Copy(&dst, src)
	if err != nil {
		t.Fatalf("copy failed: %v", err)
	}
	if !called {
		t.Fatal("expected io.Copy to use the wrapped file WriteTo fast path")
	}
	if got := dst.String(); got != "payload" {
		t.Fatalf("copied data mismatch: got %q", got)
	}
	if n != int64(len("payload")) {
		t.Fatalf("copied byte count mismatch: got %d", n)
	}
}

func TestBasePathFileForwardsReaderFrom(t *testing.T) {
	base := NewMemMapFs()
	if err := base.MkdirAll("/root", 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	var called bool
	fs := optionalIfaceFs{
		Fs: base,
		createWrap: func(file File) File {
			return &readerFromTrackingFile{File: file, called: &called}
		},
	}
	bp := NewBasePathFs(fs, "/root")

	dst, err := bp.Create("dst.txt")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	defer dst.Close()

	if _, ok := dst.(io.ReaderFrom); !ok {
		t.Fatal("BasePathFile should expose io.ReaderFrom when the wrapped file implements it")
	}

	n, err := io.Copy(dst, &slowReader{data: []byte("payload")})
	if err != nil {
		t.Fatalf("copy failed: %v", err)
	}
	if !called {
		t.Fatal("expected io.Copy to use the wrapped file ReadFrom fast path")
	}
	if n != int64(len("payload")) {
		t.Fatalf("copied byte count mismatch: got %d", n)
	}

	got, err := ReadFile(base, "/root/dst.txt")
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(got) != "payload" {
		t.Fatalf("written data mismatch: got %q", string(got))
	}
}

func TestBasePathFileCopyFallbackWithNestedBasePaths(t *testing.T) {
	base := NewMemMapFs()
	if err := base.MkdirAll("/root/nested", 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := WriteFile(base, "/root/nested/src.txt", []byte("payload"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	level1 := NewBasePathFs(base, "/root")
	level2 := NewBasePathFs(level1, "/nested")

	src, err := level2.Open("src.txt")
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	defer src.Close()

	dst, err := level2.Create("dst.txt")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		t.Fatalf("copy failed: %v", err)
	}

	if dir := filepath.Dir(src.Name()); dir != filepath.Clean(string(os.PathSeparator)) {
		t.Fatalf("nested source name leaked base path: %q", src.Name())
	}
	if dir := filepath.Dir(dst.Name()); dir != filepath.Clean(string(os.PathSeparator)) {
		t.Fatalf("nested destination name leaked base path: %q", dst.Name())
	}

	got, err := ReadFile(base, "/root/nested/dst.txt")
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(got) != "payload" {
		t.Fatalf("copied data mismatch: got %q", string(got))
	}
}
