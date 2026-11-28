package afero_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

func TestMemMapRenameDirectoryIntoDescendant(t *testing.T) {
	for _, destination := range []string{"source/child", "source/child/moved", "source/../source/child/moved"} {
		t.Run(destination, func(t *testing.T) {
			for _, fs := range []afero.Fs{afero.NewMemMapFs(), afero.NewOsFs()} {
				root := t.TempDir()
				source := filepath.Join(root, "source")
				file := filepath.Join(source, "child", "data")
				if err := fs.MkdirAll(filepath.Dir(file), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := afero.WriteFile(fs, file, []byte("keep this data"), 0o600); err != nil {
					t.Fatal(err)
				}
				target := root + string(filepath.Separator) + filepath.FromSlash(destination)
				if err := fs.Rename(source, target); err == nil {
					t.Fatalf("%s allowed moving a directory into its descendant", fs.Name())
				}
				data, err := afero.ReadFile(fs, file)
				if err != nil || string(data) != "keep this data" {
					t.Errorf(
						"%s changed the source after a rejected rename: (%q, %v)",
						fs.Name(),
						data,
						err,
					)
				}
				info, err := fs.Stat(source)
				if err != nil || !info.IsDir() {
					t.Errorf("%s lost the source directory: %v", fs.Name(), err)
				}
			}
		})
	}
}

func TestMemMapRenameDirectoryWithSimilarPrefix(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := fs.MkdirAll("source/child", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := fs.Mkdir("source2", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := afero.WriteFile(fs, "source/child/data", []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := fs.Rename("source", "source2/moved"); err != nil {
		t.Fatal(err)
	}
	data, err := afero.ReadFile(fs, "source2/moved/child/data")
	if err != nil || string(data) != "payload" {
		t.Fatalf("valid directory move returned (%q, %v)", data, err)
	}
	if _, err := fs.Stat("source"); !os.IsNotExist(err) {
		t.Fatalf("source still exists: %v", err)
	}
}

func TestMemMapRenameRootIntoDescendant(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := fs.MkdirAll("child", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := fs.Rename(string(filepath.Separator), filepath.Join("child", "moved")); err == nil {
		t.Fatal("moving the memory filesystem root into a descendant succeeded")
	}
	if _, err := fs.Stat("child"); err != nil {
		t.Fatal(err)
	}
}
