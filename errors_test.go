package afero

import (
	"errors"
	"os"
	"syscall"
	"testing"
)

func TestBrandedErrorsMatchStandardLibrary(t *testing.T) {
	if !errors.Is(ErrPermission, os.ErrPermission) {
		t.Fatal("ErrPermission should match os.ErrPermission")
	}
	if !errors.Is(ErrDeadlineExceeded, os.ErrDeadlineExceeded) {
		t.Fatal("ErrDeadlineExceeded should match os.ErrDeadlineExceeded")
	}
	if !errors.Is(ErrNotDir, syscall.ENOTDIR) {
		t.Fatal("ErrNotDir should match syscall.ENOTDIR")
	}
}
