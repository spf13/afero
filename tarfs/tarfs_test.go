// Most of the tests are stolen from the zipfs implementation
package tarfs

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
	"testing"
)

var files = []struct {
	name        string
	exists      bool
	isdir       bool
	size        int64
	content     string
	contentAt4k string
}{
	{"sub", true, true, 0, "", ""},
	{"sub/testDir2", true, true, 0, "", ""},
	{"testFile", true, false, 8192, "aaaaaaaa", "aaaabbbb"},
	{"testDir1/testFile", true, false, 8192, "bbbbbbbb", "bbbbcccc"},

	{"nonExisting", false, false, 0, "", ""},
}

var tfs *Fs

func TestMain(m *testing.M) {
	tf, err := os.Open("testdata/t.tar")
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	tfs = New(tar.NewReader(tf))
	os.Exit(m.Run())
}

func TestFsNew(t *testing.T) {

	t.Logf("+%v", tfs)

	for _, f := range files {
		e, found := tfs.files[f.name]
		if found != f.exists {
			t.Fatalf("%v exists == %v, should be %v", f.name, found, f.exists)
		}

		if !f.exists {
			continue
		}

		if e.h.Typeflag == tar.TypeDir && !f.isdir {
			t.Errorf("%v is a directory, and should not be", e.Name())
		}
	}

}

func TestFsOpen(t *testing.T) {
	for _, f := range files {
		file, err := tfs.Open(f.name)
		if (err == nil) != f.exists {
			t.Errorf("%v exists = %v, but got err = %v", file.Name(), f.exists, err)
		}

		if !f.exists {
			continue
		}

		s, err := file.Stat()
		if err != nil {
			t.Errorf("stat %v: got error '%v'", file.Name(), err)
			continue
		}

		if isdir := s.IsDir(); isdir != f.isdir {
			t.Errorf("%v directory, got: %v, expected: %v", file.Name(), isdir, f.isdir)
		}

		if size := s.Size(); size != f.size {
			t.Errorf("%v size, got: %v, expected: %v", file.Name(), size, f.size)
		}
	}
}

func TestRead(t *testing.T) {
	for _, f := range files {
		if !f.exists {
			continue
		}

		file, err := tfs.Open(f.name)
		if err != nil {
			t.Fatalf("opening %v: %v", f.name, err)
		}

		buf := make([]byte, 8)
		n, err := file.Read(buf)
		if err != nil {
			if f.isdir && (err != syscall.EISDIR) {
				t.Errorf("%v got error %v, expected EISDIR", f.name, err)
			} else if !f.isdir {
				t.Errorf("%v: %v", f.name, err)
			}
		} else if n != 8 {
			t.Errorf("%v: got %d read bytes, expected 8", f.name, n)
		} else if string(buf) != f.content {
			t.Errorf("%v: got <%s>, expected <%s>", f.name, f.content, string(buf))
		}

	}
}

func TestReadAt(t *testing.T) {
	for _, f := range files {
		if !f.exists {
			continue
		}

		file, err := tfs.Open(f.name)
		if err != nil {
			t.Fatalf("opening %v: %v", f.name, err)
		}

		buf := make([]byte, 8)
		n, err := file.ReadAt(buf, 4092)
		if err != nil {
			if f.isdir && (err != syscall.EISDIR) {
				t.Errorf("%v got error %v, expected EISDIR", f.name, err)
			} else if !f.isdir {
				t.Errorf("%v: %v", f.name, err)
			}
		} else if n != 8 {
			t.Errorf("%v: got %d read bytes, expected 8", f.name, n)
		} else if string(buf) != f.contentAt4k {
			t.Errorf("%v: got <%s>, expected <%s>", f.name, f.contentAt4k, string(buf))
		}

	}
}

func TestSeek(t *testing.T) {
	for _, f := range files {
		if !f.exists {
			continue
		}

		file, err := tfs.Open(f.name)
		if err != nil {
			t.Fatalf("opening %v: %v", f.name, err)
		}

		var tests = []struct {
			offin  int64
			whence int
			offout int64
		}{
			{0, io.SeekStart, 0},
			{10, io.SeekStart, 10},
			{1, io.SeekCurrent, 11},
			{10, io.SeekCurrent, 21},
			{0, io.SeekEnd, f.size},
			{-1, io.SeekEnd, f.size - 1},
		}

		for _, s := range tests {
			n, err := file.Seek(s.offin, s.whence)
			if err != nil {
				if f.isdir && err == syscall.EISDIR {
					continue
				}

				t.Errorf("%v: %v", f.name, err)
			}

			if n != s.offout {
				t.Errorf("%v: (off: %v, whence: %v): got %v, expected %v", f.name, s.offin, s.whence, n, s.offout)
			}
		}

	}
}

func TestName(t *testing.T) {
	for _, f := range files {
		if !f.exists {
			continue
		}

		file, err := tfs.Open(f.name)
		if err != nil {
			t.Fatalf("opening %v: %v", f.name, err)
		}

		n := file.Name()
		if n != f.name {
			t.Errorf("got: %v, expected: %v", n, f.name)
		}

	}
}

func TestClose(t *testing.T) {
	for _, f := range files {
		if !f.exists {
			continue
		}

		file, err := tfs.Open(f.name)
		if err != nil {
			t.Fatalf("opening %v: %v", f.name, err)
		}

		err = file.Close()
		if err != nil {
			t.Errorf("%v: %v", f.name, err)
		}

		err = file.Close()
		if err == nil {
			t.Errorf("%v: closing twice should return an error", f.name)
		}

		buf := make([]byte, 8)
		n, err := file.Read(buf)
		if n != 0 || err == nil {
			t.Errorf("%v: could read from a closed file", f.name)
		}

		n, err = file.ReadAt(buf, 256)
		if n != 0 || err == nil {
			t.Errorf("%v: could readAt from a closed file", f.name)
		}

		off, err := file.Seek(0, io.SeekStart)
		if off != 0 || err == nil {
			t.Errorf("%v: could seek from a closed file", f.name)
		}
	}
}

func TestOpenFile(t *testing.T) {
	for _, f := range files {
		file, err := tfs.OpenFile(f.name, os.O_RDONLY, 0400)
		if !f.exists {
			if !errors.Is(err, syscall.ENOENT) {
				t.Errorf("%v: got %v, expected%v", f.name, err, syscall.ENOENT)
			}
			file.Close()

			continue
		}

		if err != nil {
			t.Errorf("%v: %v", f.name, err)
		}
		file.Close()

		file, err = tfs.OpenFile(f.name, os.O_CREATE, 0600)
		if !errors.Is(err, syscall.EPERM) {
			t.Errorf("%v: open for write: got %v, expected %v", f.name, err, syscall.EPERM)
		}
		file.Close()

	}
}
