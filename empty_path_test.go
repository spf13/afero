package afero

import (
	"errors"
	"os"
	"testing"
)

func TestWriteReadFileEmptyPath(t *testing.T) {
	fs := NewMemMapFs()
	if err := WriteFile(fs, "", []byte("x"), 0o644); err == nil {
		t.Fatal("expected write error")
	} else if !errors.Is(err, os.ErrInvalid) {
		// PathError may wrap
		var pe *os.PathError
		if !errors.As(err, &pe) {
			t.Fatalf("got %v", err)
		}
	}
	if _, err := ReadFile(fs, ""); err == nil {
		t.Fatal("expected read error")
	}
}
