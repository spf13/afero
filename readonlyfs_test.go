package afero

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TestMkdirAllReadonly(t *testing.T) {
	base := &MemMapFs{}
	ro := &ReadOnlyFs{source: base}

	base.MkdirAll("/home/test", 0o777)
	if err := ro.MkdirAll("/home/test", 0o777); err != nil {
		t.Errorf("Failed to MkdirAll on existing path in ReadOnlyFs: %s", err)
	}

	if err := ro.MkdirAll("/home/test/newdir", 0o777); err == nil {
		t.Error("Creating new dir with MkdirAll on ReadOnlyFs should fail but returned nil")
	}

	base.Create("/home/test/file")
	if err := ro.MkdirAll("/home/test/file", 0o777); err == nil {
		t.Error("Creating new dir with MkdirAll on ReadOnlyFs where a file already exists should fail but returned nil")
	}
}

func TestReadOnlyFsMutations(t *testing.T) {
	base := &MemMapFs{}
	base.MkdirAll("/home/test", 0o777)
	base.Create("/home/test/file")
	ro := NewReadOnlyFs(base)

	if ro.Name() != "ReadOnlyFilter" {
		t.Errorf("Expected Name() to be %q, got %q", "ReadOnlyFilter", ro.Name())
	}

	if _, err := ro.Create("/home/test/newfile"); err != syscall.EPERM {
		t.Errorf("Expected syscall.EPERM on Create, got %v", err)
	}

	if err := ro.Remove("/home/test/file"); err != syscall.EPERM {
		t.Errorf("Expected syscall.EPERM on Remove, got %v", err)
	}

	if err := ro.RemoveAll("/home/test"); err != syscall.EPERM {
		t.Errorf("Expected syscall.EPERM on RemoveAll, got %v", err)
	}

	if err := ro.Rename("/home/test/file", "/home/test/renamed"); err != syscall.EPERM {
		t.Errorf("Expected syscall.EPERM on Rename, got %v", err)
	}

	if err := ro.Chmod("/home/test/file", 0o644); err != syscall.EPERM {
		t.Errorf("Expected syscall.EPERM on Chmod, got %v", err)
	}

	if err := ro.Chown("/home/test/file", 1000, 1000); err != syscall.EPERM {
		t.Errorf("Expected syscall.EPERM on Chown, got %v", err)
	}

	now := time.Now()
	if err := ro.Chtimes("/home/test/file", now, now); err != syscall.EPERM {
		t.Errorf("Expected syscall.EPERM on Chtimes, got %v", err)
	}

	if err := ro.Mkdir("/home/test/newdir", 0o755); err != syscall.EPERM {
		t.Errorf("Expected syscall.EPERM on Mkdir, got %v", err)
	}

	if _, err := ro.OpenFile("/home/test/file", os.O_WRONLY, 0o644); err != syscall.EPERM {
		t.Errorf("Expected syscall.EPERM on OpenFile with O_WRONLY, got %v", err)
	}

	f, err := ro.Open("/home/test/file")
	if err != nil {
		t.Errorf("Unexpected error on Open: %v", err)
	} else {
		f.Close()
	}
}
