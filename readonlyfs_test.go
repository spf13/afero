package afero

import (
	"testing"
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
