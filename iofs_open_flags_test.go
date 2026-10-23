//go:build go1.16
// +build go1.16

package afero

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"
)

type openFlagsFS struct {
	fs.FS
	opens int
}

func (f *openFlagsFS) Open(name string) (fs.File, error) {
	f.opens++
	return f.FS.Open(name)
}

func TestFromIOFSOpenFileWriteFlags(t *testing.T) {
	for _, tc := range []struct {
		name string
		flag int
	}{
		{"write_only", os.O_WRONLY},
		{"read_write", os.O_RDWR},
		{"append", os.O_APPEND},
		{"create", os.O_CREATE},
		{"truncate", os.O_TRUNC},
		{"create_exclusive", os.O_CREATE | os.O_EXCL},
		{"write_truncate", os.O_WRONLY | os.O_TRUNC},
		{"read_write_create", os.O_RDWR | os.O_CREATE},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, name := range []string{"file", "missing"} {
				t.Run(name, func(t *testing.T) {
					base := &openFlagsFS{FS: fstest.MapFS{
						"file": &fstest.MapFile{Data: []byte("preserved")},
					}}
					f, err := (FromIOFS{FS: base}).OpenFile(name, tc.flag, 0600)
					if f != nil {
						f.Close()
						t.Error("write-intent OpenFile returned a file")
					}
					if !errors.Is(err, fs.ErrPermission) {
						t.Errorf("OpenFile returned %v; want ErrPermission", err)
					}
					var pe *fs.PathError
					if !errors.As(err, &pe) || pe.Op != "openfile" || pe.Path != name {
						t.Errorf("OpenFile error = %#v; want PathError for %q", err, name)
					}
					if base.opens != 0 {
						t.Errorf("underlying Open called %d times for write flags", base.opens)
					}
				})
			}
		})
	}
}

func TestFromIOFSOpenFileReadFlags(t *testing.T) {
	for _, tc := range []struct {
		name string
		flag int
	}{
		{"read_only", os.O_RDONLY},
		{"sync_without_write", os.O_RDONLY | os.O_SYNC},
		{"exclusive_without_create", os.O_RDONLY | os.O_EXCL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := &openFlagsFS{FS: fstest.MapFS{
				"file": &fstest.MapFile{Data: []byte("preserved")},
			}}
			adapter := FromIOFS{FS: base}
			f, err := adapter.OpenFile("file", tc.flag, 0600)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			data, err := io.ReadAll(f)
			if err != nil || string(data) != "preserved" || base.opens != 1 {
				t.Fatalf("read = %q, error = %v, opens = %d", data, err, base.opens)
			}
			if _, err := f.Write([]byte("changed")); !errors.Is(err, fs.ErrPermission) {
				t.Errorf("read-only file Write returned %v; want ErrPermission", err)
			}
			missing, err := adapter.OpenFile("missing", tc.flag, 0600)
			if missing != nil || !errors.Is(err, fs.ErrNotExist) || base.opens != 2 {
				t.Errorf("missing = %v, error = %v, opens = %d", missing, err, base.opens)
			}
		})
	}
}
