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

// https://github.com/spf13/afero/issues/713
func TestCopyOnWriteFsOpenFileOExclDoesNotCopyToLayer(t *testing.T) {
	base := NewMemMapFs()
	layer := NewMemMapFs()
	if err := WriteFile(base, "config.txt", []byte("initial"), 0o644); err != nil {
		t.Fatal(err)
	}
	fs := NewCopyOnWriteFs(base, layer)

	f, err := fs.OpenFile("config.txt", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if f != nil {
		defer f.Close()
	}
	if err == nil || !os.IsExist(err) {
		t.Fatalf("expected an already exists error, got %v", err)
	}

	// The failed exclusive create must not have copied the base file
	// into the layer.
	if _, err := layer.Stat("config.txt"); err == nil {
		t.Fatal("expected config.txt to be absent from the layer")
	}

	// Later changes to the base file must remain visible through the union.
	if err := WriteFile(base, "config.txt", []byte("updated"), 0o644); err != nil {
		t.Fatal(err)
	}
	contents, err := ReadFile(fs, "config.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "updated" {
		t.Fatalf("expected %q, got %q", "updated", contents)
	}
}
