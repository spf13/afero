package afero

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyOnWrite(t *testing.T) {
	osFs := NewOsFs()
	writeDir, err := TempDir(osFs, "", "copy-on-write-test")
	if err != nil {
		t.Fatal("error creating tempDir", err)
	}
	defer osFs.RemoveAll(writeDir)

	compositeFs := NewCopyOnWriteFs(NewReadOnlyFs(NewOsFs()), osFs)

	dir := filepath.Join(writeDir, "some/path")

	err = compositeFs.MkdirAll(dir, 0o744)
	if err != nil {
		t.Fatal(err)
	}
	_, err = compositeFs.Create(filepath.Join(dir, "newfile"))
	if err != nil {
		t.Fatal(err)
	}

	// https://github.com/spf13/afero/issues/189
	// We want the composite file system to behave like the OS file system
	// on Mkdir and MkdirAll
	for _, fs := range []Fs{osFs, compositeFs} {
		err = fs.Mkdir(dir, 0o744)
		if err == nil || !os.IsExist(err) {
			t.Errorf("Mkdir: Got %q for %T", err, fs)
		}

		// MkdirAll does not return an error when the directory already exists
		err = fs.MkdirAll(dir, 0o744)
		if err != nil {
			t.Errorf("MkdirAll:  Got %q for %T", err, fs)
		}

	}
}

func TestCopyOnWriteFileInMemMapBase(t *testing.T) {
	base := &MemMapFs{}
	layer := &MemMapFs{}

	if err := WriteFile(base, "base.txt", []byte("base"), 0o755); err != nil {
		t.Fatalf("Failed to write file: %s", err)
	}

	ufs := NewCopyOnWriteFs(base, layer)

	_, err := ufs.Stat("base.txt")
	if err != nil {
		t.Fatal(err)
	}
}

// https://github.com/spf13/afero/issues/712
func TestCopyOnWriteFsRemoveAllReportsBaseOnlyPaths(t *testing.T) {
	base := NewMemMapFs()
	layer := NewMemMapFs()
	if err := base.MkdirAll("base-only", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(base, "base-only/file.txt", []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	fs := NewCopyOnWriteFs(base, layer)

	// A directory that only exists in the base cannot be removed; the
	// call must not report success, consistent with Remove.
	if err := fs.RemoveAll("base-only"); err == nil {
		t.Fatal("expected an error removing a base-only directory")
	}
	exists, err := Exists(fs, "base-only/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("expected base-only/file.txt to still be visible")
	}

	// Paths in the layer are still removed normally.
	if err := WriteFile(layer, "layer-only/file.txt", []byte("gone"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fs.RemoveAll("layer-only"); err != nil {
		t.Fatalf("expected success removing a layer-only directory, got %v", err)
	}
	if exists, _ := Exists(fs, "layer-only/file.txt"); exists {
		t.Fatal("expected layer-only/file.txt to be gone")
	}
}
