package afero

import (
	"errors"
	"os"
	"testing"
)

func TestMemMapFsStatEmptyPath(t *testing.T) {
	t.Parallel()

	_, err := NewMemMapFs().Stat("")
	if err == nil {
		t.Fatal("expected error for empty path, got nil")
	}

	var pathErr *os.PathError
	if !errors.As(err, &pathErr) {
		t.Fatalf("expected *os.PathError, got %T", err)
	}
	if pathErr.Op != "stat" {
		t.Fatalf("expected Op stat, got %q", pathErr.Op)
	}

	_, err = os.Stat("")
	if err == nil {
		t.Fatal(`os.Stat("") should fail for comparison`)
	}
}

func TestMemMapFsStatRootName(t *testing.T) {
	t.Parallel()

	info, err := NewMemMapFs().Stat("/")
	if err != nil {
		t.Fatal(err)
	}
	if info.Name() != FilePathSeparator {
		t.Fatalf("expected root name %q, got %q", FilePathSeparator, info.Name())
	}
}
