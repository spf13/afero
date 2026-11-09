package afero

import (
	"os"
	"path/filepath"
	"syscall"
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

func TestCopyOnWriteFsRemoveAllBaseOnlyDirectory(t *testing.T) {
	base := NewMemMapFs()
	layer := NewMemMapFs()
	if err := base.MkdirAll("base-only", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(base, "base-only/file.txt", []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	fs := NewCopyOnWriteFs(base, layer)

	err := fs.RemoveAll("base-only")
	if err != syscall.EPERM {
		t.Fatalf("RemoveAll base-only directory: expected EPERM, got %v", err)
	}
	exists, statErr := Exists(fs, "base-only/file.txt")
	if statErr != nil {
		t.Fatal(statErr)
	}
	if !exists {
		t.Fatal("base-only directory should remain visible after a failed RemoveAll")
	}

	if err := fs.RemoveAll("missing"); err != nil {
		t.Fatalf("RemoveAll missing path: expected nil, got %v", err)
	}

	if err := layer.MkdirAll("layer-only", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(layer, "layer-only/file.txt", []byte("gone"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fs.RemoveAll("layer-only"); err != nil {
		t.Fatalf("RemoveAll layer-only directory: %v", err)
	}
	exists, statErr = Exists(fs, "layer-only")
	if statErr != nil {
		t.Fatal(statErr)
	}
	if exists {
		t.Fatal("layer-only directory should be removed")
	}

	if err := base.MkdirAll("shared", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(base, "shared/base.txt", []byte("base"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(fs, "shared/layer.txt", []byte("layer"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fs.RemoveAll("shared"); err != nil {
		t.Fatalf("RemoveAll overlay+base directory: expected nil, got %v", err)
	}
	if _, err := layer.Stat("shared"); !os.IsNotExist(err) {
		t.Fatalf("expected overlay copy of shared to be gone, Stat error: %v", err)
	}
	exists, statErr = Exists(fs, "shared/base.txt")
	if statErr != nil {
		t.Fatal(statErr)
	}
	if !exists {
		t.Fatal("base copy of shared should remain visible")
	}
}
