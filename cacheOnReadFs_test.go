package afero

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCacheOnReadFsWriteFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache-on-read-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	base := NewBasePathFs(NewOsFs(), tempDir)
	layer := NewMemMapFs()
	fs := NewCacheOnReadFs(base, layer, 0)

	content := []byte("content here")
	target := filepath.Join("sub", "file.txt")
	err = fs.MkdirAll("sub", 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Issue #390: WriteFile on CacheOnReadFs failed with "bad file descriptor"
	err = WriteFile(fs, target, content, 0o644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	for _, subFs := range []Fs{fs, base, layer} {
		got, err := ReadFile(subFs, target)
		if err != nil || !bytes.Equal(got, content) {
			t.Errorf("ReadFile(%s) = %q, %v; want %q, nil", subFs.Name(), got, err, content)
		}
	}
}

func TestCacheOnReadFsWriteExistingFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache-on-read-test-existing")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	base := NewBasePathFs(NewOsFs(), tempDir)
	layer := NewMemMapFs()

	initialContent := []byte("initial content")
	if err := WriteFile(base, "existing.txt", initialContent, 0o644); err != nil {
		t.Fatal(err)
	}

	fs := NewCacheOnReadFs(base, layer, 0)

	newContent := []byte("updated content")
	if err := WriteFile(fs, "existing.txt", newContent, 0o644); err != nil {
		t.Fatalf("WriteFile failed on existing file: %v", err)
	}

	for _, subFs := range []Fs{fs, base, layer} {
		got, err := ReadFile(subFs, "existing.txt")
		if err != nil || !bytes.Equal(got, newContent) {
			t.Errorf("ReadFile(%s) = %q, %v; want %q, nil", subFs.Name(), got, err, newContent)
		}
	}
}
