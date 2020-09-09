// Most of the tests are stolen from the zipfs implementation
package tarfs

import (
	"archive/tar"
	"fmt"
	"os"
	"testing"
)

var files = []struct {
	name   string
	exists bool
	isdir  bool
	size   int64
}{
	{"sub", true, true, 0},
	{"sub/testDir2", true, true, 0},
	{"testFile", true, false, 8192},
	{"testDir1/testFile", true, false, 8192},

	{"nonExisting", false, false, 0},
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

func TestFileOps(t *testing.T) {
	for _, f := range files {
		if !f.exists {
			continue
		}

		file, err := tfs.Open(f.name)
		if err != nil {
			t.Fatalf("opening %v: %v", f.name, err)
		}

		buf := make([]byte, 8)
		if n, err := file.ReadAt(buf, 4092); err != nil {
			t.Error(err)
		} else if n != 8 {
			t.Errorf("expected to read 8 bytes, got %d", n)
		} else if string(buf) != "aaaabbbb" {
			t.Errorf("expected to get <aaaabbbb>, got <%s>", string(buf))
		}

	}
}
