package afero

import (
	"os"
	"testing"
	"time"
)

func checkForErrPermission(t *testing.T, err error) {
	t.Helper()
	if err == nil || !os.IsPermission(err) {
		t.Errorf("Expected err !=nil && err == ErrPermission, got %[1]T (%[1]v)", err)
	}
}

// Make sure that the ReadOnlyFs filter returns errors that can be
// checked with os.IsPermission
func TestReadOnlyFsErrPermission(t *testing.T) {
	fs := NewReadOnlyFs(NewMemMapFs())

	_, err := fs.Create("test")
	checkForErrPermission(t, err)
	checkForErrPermission(t, fs.Chtimes("test", time.Now(), time.Now()))
	checkForErrPermission(t, fs.Chmod("test", os.ModePerm))
	checkForErrPermission(t, fs.Chown("test", 0, 0))
	checkForErrPermission(t, fs.Mkdir("test", os.ModePerm))
	checkForErrPermission(t, fs.MkdirAll("test", os.ModePerm))
	_, err = fs.OpenFile("test", os.O_CREATE, os.ModePerm)
	checkForErrPermission(t, err)
	checkForErrPermission(t, fs.Remove("test"))
	checkForErrPermission(t, fs.RemoveAll("test"))
	checkForErrPermission(t, fs.Rename("test", "test"))

}
