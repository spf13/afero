package afero

import (
	"bytes"
	"io"
	"regexp"
	"testing"
)

// --- tracking harness (same shape as basepath_copy_test.go's optionalIfaceFs) ---

type sibOptionalIfaceFs struct {
	Fs
	openWrap func(File) File
}

func (f sibOptionalIfaceFs) Open(name string) (File, error) {
	file, err := f.Fs.Open(name)
	if err != nil {
		return nil, err
	}
	if f.openWrap != nil {
		return f.openWrap(file), nil
	}
	return file, nil
}

type sibWriterToTrackingFile struct {
	File
	called *bool
}

func (f *sibWriterToTrackingFile) WriteTo(w io.Writer) (int64, error) {
	*f.called = true
	return io.Copy(w, f.File)
}

// TestRegexpFile_WriteTo_FastPath asserts that io.Copy through a RegexpFs
// observes the underlying file's io.WriterTo fast path. This is the sibling of
// PR #577 (BasePathFile). On the UNFIXED tree it FAILS (fast path hidden).
func TestRegexpFile_WriteTo_FastPath(t *testing.T) {
	const payload = "hello-regexp-fastpath"

	base := &MemMapFs{}
	if err := WriteFile(base, "match.txt", []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}

	called := false
	tracking := sibOptionalIfaceFs{
		Fs: base,
		openWrap: func(file File) File {
			return &sibWriterToTrackingFile{File: file, called: &called}
		},
	}

	rfs := NewRegexpFs(tracking, regexp.MustCompile(`\.txt$`))

	f, err := rfs.Open("match.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var buf bytes.Buffer
	n, err := io.Copy(&buf, f)
	if err != nil {
		t.Fatal(err)
	}
	if int(n) != len(payload) || buf.String() != payload {
		t.Fatalf("copy mismatch: n=%d got=%q", n, buf.String())
	}

	if !called {
		t.Fatalf("io.WriterTo fast path NOT taken through RegexpFs (bug reproduced): underlying WriteTo never called")
	}
}

// NOTE on the ReadFrom direction: RegexpFs.Create and RegexpFs.OpenFile return
// the source file UNWRAPPED (they do not produce a *RegexpFile). Only
// RegexpFs.Open wraps in *RegexpFile. Therefore the io.ReaderFrom-hiding bug is
// not reachable through RegexpFs today, and a ReadFrom assertion would be
// vacuous. The sibling fix mirrors PR #577's ReadFrom for interface parity, but
// the only demonstrable bug is WriteTo (tested above).
