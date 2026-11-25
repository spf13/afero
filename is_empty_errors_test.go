package afero

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestIsEmptyPreservesStatErrors(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires Unix directory permissions")
	}
	dir := filepath.Join(t.TempDir(), "restricted")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "file")
	if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	source := NewOsFs()
	if _, err := source.Stat(path); !errors.Is(err, os.ErrPermission) {
		t.Skipf("permissions not enforced: %v", err)
	}
	for name, check := range map[string]func(string) (bool, error){
		"function": func(path string) (bool, error) { return IsEmpty(source, path) },
		"method":   (Afero{Fs: source}).IsEmpty,
	} {
		t.Run(name, func(t *testing.T) {
			empty, err := check(path)
			if empty || !errors.Is(err, os.ErrPermission) {
				t.Fatalf("IsEmpty = %v, %v; want permission error", empty, err)
			}
		})
	}
}
