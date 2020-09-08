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
