package afero

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestMemMapCreateDirectory(t *testing.T) {
	for _, tc := range []struct {
		name      string
		path      string
		populated bool
	}{
		{name: "empty", path: "directory"},
		{name: "nonempty", path: "directory", populated: true},
		{name: "root", path: ".", populated: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := NewMemMapFs()
			if err := fs.MkdirAll(tc.path, 0o755); err != nil {
				t.Fatal(err)
			}
			child := filepath.Join(tc.path, "child")
			if tc.populated {
				if err := WriteFile(fs, child, []byte("preserved"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			before, err := fs.Stat(tc.path)
			if err != nil {
				t.Fatal(err)
			}
			beforeMode, beforeModTime := before.Mode(), before.ModTime()
			f, err := fs.Create(tc.path)
			if f != nil {
				f.Close()
				t.Error("Create(directory) returned a file")
			}
			var pathErr *os.PathError
			if !errors.As(err, &pathErr) || !errors.Is(err, syscall.EISDIR) {
				t.Errorf("Create(directory) = %v; want PathError wrapping EISDIR", err)
			} else if pathErr.Op != "create" || pathErr.Path != normalizePath(tc.path) {
				t.Errorf("unexpected PathError: %+v", pathErr)
			}
			after, err := fs.Stat(tc.path)
			if err != nil {
				t.Fatal(err)
			}
			if !after.IsDir() || after.Mode() != beforeMode || !after.ModTime().Equal(beforeModTime) {
				t.Errorf("Create changed directory metadata: mode=%v modtime=%v", after.Mode(), after.ModTime())
			}
			entries, err := ReadDir(fs, tc.path)
			if err != nil {
				t.Fatal(err)
			}
			wantEntries := 0
			if tc.populated {
				wantEntries = 1
				data, err := ReadFile(fs, child)
				if err != nil || string(data) != "preserved" {
					t.Errorf("child content = %q, %v", data, err)
				}
			}
			if len(entries) != wantEntries || (tc.populated && entries[0].Name() != "child") {
				t.Errorf("directory entries = %v; want %d children", entries, wantEntries)
			}
			// A failed Create must release the filesystem lock.
			f, err = fs.Create(filepath.Join(tc.path, "new-file"))
			if err != nil {
				t.Fatal(err)
			}
			f.Close()
		})
	}
}

func TestMemMapCreateTruncatesFile(t *testing.T) {
	fs := NewMemMapFs()
	f, err := fs.Create("file")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("old content"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	f, err = fs.Create("file")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := ReadFile(fs, "file")
	if err != nil || len(data) != 0 {
		t.Errorf("Create did not truncate the file: %q, %v", data, err)
	}
}
