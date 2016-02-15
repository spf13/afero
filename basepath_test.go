package afero

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBasePath(t *testing.T) {
	baseFs := &MemMapFs{}
	baseFs.MkdirAll("/base/path/tmp", 0777)
	bp := NewBasePathFs(baseFs, "/base/path")

	if _, err := bp.Create("/tmp/foo"); err != nil {
		t.Errorf("Failed to set real path")
	}

	if fh, err := bp.Create("../tmp/bar"); err == nil {
		t.Errorf("succeeded in creating %s ...", fh.Name())
	}
}

func TestBasePathRoot(t *testing.T) {
	baseFs := &MemMapFs{}
	baseFs.MkdirAll("/base/path/foo/baz", 0777)
	baseFs.MkdirAll("/base/path/boo/", 0777)
	bp := NewBasePathFs(baseFs, "/base/path")

	rd, err := ReadDir(bp, string(os.PathSeparator))

	if len(rd) != 2 {
		t.Errorf("base path doesn't respect root")
	}

	if err != nil {
		t.Error(err)
	}
}

func TestRealPath(t *testing.T) {
	fs := NewOsFs()
	baseDir, err := TempDir(fs, "", "base")
	if err != nil {
		t.Fatal("error creating tempDir", err)
	}
	defer fs.RemoveAll(baseDir)
	anotherDir, err := TempDir(fs, "", "another")
	if err != nil {
		t.Fatal("error creating tempDir", err)
	}
	defer fs.RemoveAll(anotherDir)

	bp := NewBasePathFs(fs, baseDir).(*BasePathFs)

	subDir := filepath.Join(baseDir, "s1")

	realPath, err := bp.RealPath("/s1")

	if err != nil {
		t.Errorf("Got error %s", err)
	}

	if realPath != subDir {
		t.Errorf("Expected \n%s got \n%s", subDir, realPath)
	}

	if runtime.GOOS == "windows" {
		_, err = bp.RealPath(anotherDir)

		if err == nil {
			t.Errorf("Expected error")
		}

	} else {
		// on *nix we have no way of just looking at the path and tell that anotherDir
		// is not inside the base file system.
		// The user will receive an os.ErrNotExist later.
		surrealPath, err := bp.RealPath(anotherDir)

		if err != nil {
			t.Errorf("Got error %s", err)
		}

		excpected := filepath.Join(baseDir, anotherDir)

		if surrealPath != excpected {
			t.Errorf("Expected \n%s got \n%s", excpected, surrealPath)
		}
	}

}
