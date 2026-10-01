package afero

import (
	"os"
	"path/filepath"
	"regexp"
	"syscall"
	"testing"
)

// controlSucceeds runs rc.Control and fails the test if either the Control
// call itself or the callback-reported syscall error is non-nil, and if the
// file descriptor handed to the callback is not valid (> 0).
func controlSucceeds(t *testing.T, rc syscall.RawConn) {
	t.Helper()
	var fd uintptr
	var innerErr error
	if err := rc.Control(func(descriptor uintptr) {
		fd = descriptor
	}); err != nil {
		innerErr = err
	}
	if innerErr != nil {
		t.Fatalf("rc.Control returned error: %v", innerErr)
	}
	if fd == 0 {
		t.Fatalf("rc.Control handed back an invalid (zero) file descriptor")
	}
}

func TestBasePathFileSyscallConn(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("payload"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	bp := NewBasePathFs(NewOsFs(), dir)
	f, err := bp.Open("f.txt")
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	defer f.Close()

	sc, ok := f.(syscall.Conn)
	if !ok {
		t.Fatal("BasePathFile wrapping *os.File should implement syscall.Conn")
	}
	rc, err := sc.SyscallConn()
	if err != nil {
		t.Fatalf("SyscallConn returned error: %v", err)
	}
	controlSucceeds(t, rc)
}

func TestRegexpFileSyscallConn(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("payload"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	rfs := NewRegexpFs(NewOsFs(), regexp.MustCompile(`\.txt$`))
	f, err := rfs.Open(filepath.Join(dir, "f.txt"))
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	defer f.Close()

	sc, ok := f.(syscall.Conn)
	if !ok {
		t.Fatal("RegexpFile wrapping *os.File should implement syscall.Conn")
	}
	rc, err := sc.SyscallConn()
	if err != nil {
		t.Fatalf("SyscallConn returned error: %v", err)
	}
	controlSucceeds(t, rc)
}

// Negative control: when the wrapped file does NOT implement syscall.Conn
// (e.g. an in-memory file), SyscallConn must return a descriptive error
// instead of panicking or silently succeeding.
func TestBasePathFileSyscallConnUnsupported(t *testing.T) {
	mem := NewMemMapFs()
	if err := WriteFile(mem, "/f.txt", []byte("payload"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	bp := NewBasePathFs(mem, "/")
	f, err := bp.Open("f.txt")
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	defer f.Close()

	sc, ok := f.(syscall.Conn)
	if !ok {
		t.Fatal("BasePathFile must always expose the syscall.Conn method (even if it errors)")
	}
	if _, err := sc.SyscallConn(); err == nil {
		t.Fatal("expected SyscallConn to return an error for a non-syscall.Conn backing file")
	}
}

func TestRegexpFileSyscallConnUnsupported(t *testing.T) {
	mem := NewMemMapFs()
	if err := WriteFile(mem, "/f.txt", []byte("payload"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	rfs := NewRegexpFs(mem, regexp.MustCompile(`\.txt$`))
	f, err := rfs.Open("/f.txt")
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	defer f.Close()

	sc, ok := f.(syscall.Conn)
	if !ok {
		t.Fatal("RegexpFile must always expose the syscall.Conn method (even if it errors)")
	}
	if _, err := sc.SyscallConn(); err == nil {
		t.Fatal("expected SyscallConn to return an error for a non-syscall.Conn backing file")
	}
}
