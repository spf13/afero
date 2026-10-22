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

func TestCopyOnWrite_PreservesPermissions_MemMap(t *testing.T) {
	base := &MemMapFs{}
	layer := &MemMapFs{}

	if err := base.Mkdir("sub", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(base, "sub/secret.txt", []byte("secret-data"), 0o600); err != nil {
		t.Fatal(err)
	}

	ufs := NewCopyOnWriteFs(base, layer)

	// Open for writing to trigger copy-up to layer
	f, err := ufs.OpenFile("sub/secret.txt", os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("append"); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	// Verify layer directory mode is 0700
	dirFi, err := layer.Stat("sub")
	if err != nil {
		t.Fatalf("layer.Stat(sub) failed: %v", err)
	}
	if dirFi.Mode().Perm() != 0o700 {
		t.Errorf("layer directory perm = %o, want 0700", dirFi.Mode().Perm())
	}

	// Verify layer file mode is 0600
	fileFi, err := layer.Stat("sub/secret.txt")
	if err != nil {
		t.Fatalf("layer.Stat(sub/secret.txt) failed: %v", err)
	}
	if fileFi.Mode().Perm() != 0o600 {
		t.Errorf("layer file perm = %o, want 0600", fileFi.Mode().Perm())
	}
}

func TestCopyOnWrite_PreservesPermissions_OsFs(t *testing.T) {
	osFs := NewOsFs()
	tmpDir, err := TempDir(osFs, "", "cow-perm-test")
	if err != nil {
		t.Fatal(err)
	}
	defer osFs.RemoveAll(tmpDir)

	baseDir := filepath.Join(tmpDir, "base")
	layerDir := filepath.Join(tmpDir, "layer")

	if err := osFs.Mkdir(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := osFs.Mkdir(layerDir, 0o755); err != nil {
		t.Fatal(err)
	}

	base := NewBasePathFs(osFs, baseDir)
	layer := NewBasePathFs(osFs, layerDir)

	if err := base.Mkdir("sub", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(base, "sub/secret.txt", []byte("secret-data"), 0o600); err != nil {
		t.Fatal(err)
	}

	ufs := NewCopyOnWriteFs(base, layer)

	// Trigger copy-up by opening for write
	f, err := ufs.OpenFile("sub/secret.txt", os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("append"); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	// Verify layer directory mode is 0700
	dirFi, err := layer.Stat("sub")
	if err != nil {
		t.Fatalf("layer.Stat(sub) failed: %v", err)
	}
	if dirFi.Mode().Perm() != 0o700 {
		t.Errorf("layer directory perm = %o, want 0700", dirFi.Mode().Perm())
	}

	// Verify layer file mode is 0600
	fileFi, err := layer.Stat("sub/secret.txt")
	if err != nil {
		t.Fatalf("layer.Stat(sub/secret.txt) failed: %v", err)
	}
	if fileFi.Mode().Perm() != 0o600 {
		t.Errorf("layer file perm = %o, want 0600", fileFi.Mode().Perm())
	}
}
