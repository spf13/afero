package afero

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPredicateFs(t *testing.T) {
	mfs := &MemMapFs{}

	txtExts := func(name string) bool {
		return strings.HasSuffix(name, ".txt")
	}

	nonEmpty := func(name string) bool {
		fi, err := mfs.Stat(name)
		if err != nil {
			t.Errorf("Got unexpected Stat err %v", err)
		}
		// Note: If you use this rule, you cannot create any files on the fs
		return fi.Size() > 0
	}

	inHiddenDir := func(path string) bool {
		return strings.HasSuffix(filepath.Dir(path), ".hidden")
	}

	pred := func(path string) bool {
		return nonEmpty(path) && txtExts(path) && !inHiddenDir(path)
	}

	fs := &PredicateFs{pred: pred, source: mfs}

	mfs.MkdirAll("/dir/sub/.hidden", 0777)
	for _, name := range []string{"file.txt", "file.html", "empty.txt"} {
		for _, dir := range []string{"/dir/", "/dir/sub/", "/dir/sub/.hidden/"} {
			fh, _ := mfs.Create(dir + name)

			if !strings.HasPrefix(name, "empty") {
				fh.WriteString("file content")
			}

			fh.Close()
		}
	}

	files, _ := ReadDir(fs, "/dir")

	if len(files) != 2 { // file.txt, sub
		t.Errorf("Got wrong number of files: %#v", files)
	}

	f, _ := fs.Open("/dir/sub")
	names, _ := f.Readdirnames(-1)
	if len(names) != 2 {
		// file.txt, .hidden (dirs are not filtered)
		t.Errorf("Got wrong number of names: %v", names)
	}

	hiddenFiles, _ := ReadDir(fs, "/dir/sub/.hidden")
	if len(hiddenFiles) != 0 {
		t.Errorf("Got wrong number of names: %v", hiddenFiles)
	}
}
