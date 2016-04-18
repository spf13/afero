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

func TestNestedBasePaths(t *testing.T) {
	type dirSpec struct {
		Dir1, Dir2, Dir3 string
	}
	dirSpecs := []dirSpec{
		dirSpec{Dir1: "/", Dir2: "/", Dir3: "/"},
		dirSpec{Dir1: "/", Dir2: "/path2", Dir3: "/"},
		dirSpec{Dir1: "/path1/dir", Dir2: "/path2/dir/", Dir3: "/path3/dir"},
	}

	for _, ds := range dirSpecs {
		memFs := NewMemMapFs()
		level1Fs := NewBasePathFs(memFs, ds.Dir1)
		level2Fs := NewBasePathFs(level1Fs, ds.Dir2)
		level3Fs := NewBasePathFs(level2Fs, ds.Dir3)

		type spec struct {
			BaseFs       Fs
			FileName     string
			ExpectedPath string
		}
		specs := []spec{
			spec{BaseFs: level3Fs, FileName: "f.txt", ExpectedPath: filepath.Join(ds.Dir1, ds.Dir2, ds.Dir3, "f.txt")},
			spec{BaseFs: level2Fs, FileName: "f.txt", ExpectedPath: filepath.Join(ds.Dir1, ds.Dir2, "f.txt")},
			spec{BaseFs: level1Fs, FileName: "f.txt", ExpectedPath: filepath.Join(ds.Dir1, "f.txt")},
		}

		for _, s := range specs {
			if actualPath, err := s.BaseFs.(*BasePathFs).fullPath(s.FileName); err != nil {
				t.Errorf("Got error %s", err.Error())
			} else if actualPath != s.ExpectedPath {
				t.Errorf("Expected \n%s got \n%s", s.ExpectedPath, actualPath)
			}
		}
	}
}
