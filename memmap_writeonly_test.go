package afero_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

func TestMemMapWriteOnlyHandleRejectsReads(t *testing.T) {
	for _, flag := range []int{os.O_WRONLY, os.O_WRONLY | os.O_APPEND, os.O_WRONLY | os.O_CREATE} {
		for _, kind := range []string{"Read", "ReadAt"} {
			t.Run(fmt.Sprintf("%s/%d", kind, flag), func(t *testing.T) {
				for _, fs := range []afero.Fs{afero.NewMemMapFs(), afero.NewOsFs()} {
					name := filepath.Join(t.TempDir(), "data")
					if err := afero.WriteFile(fs, name, []byte("payload"), 0o600); err != nil {
						t.Fatal(err)
					}
					f, err := fs.OpenFile(name, flag, 0o600)
					if err != nil {
						t.Fatal(err)
					}
					if _, err := f.Seek(0, io.SeekStart); err != nil {
						t.Fatal(err)
					}
					buf := make([]byte, 3)
					var n int
					if kind == "Read" {
						n, err = f.Read(buf)
					} else {
						n, err = f.ReadAt(buf, 0)
					}
					if n != 0 || err == nil {
						t.Errorf("%s %s: Read returned (%d, %v), want zero bytes and an error",
							fs.Name(), kind, n, err)
					}
					if _, err := f.Write([]byte("new")); err != nil {
						t.Errorf("%s: write-only handle cannot write: %v", fs.Name(), err)
					}
					if err := f.Close(); err != nil {
						t.Fatal(err)
					}
				}
			})
		}
	}
}

func TestMemMapReadableHandles(t *testing.T) {
	for _, flag := range []int{os.O_RDONLY, os.O_RDWR} {
		fs := afero.NewMemMapFs()
		if err := afero.WriteFile(fs, "data", []byte("payload"), 0o600); err != nil {
			t.Fatal(err)
		}
		f, err := fs.OpenFile("data", flag, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(f)
		if err != nil || string(data) != "payload" {
			t.Errorf("flag %d: ReadAll = (%q, %v)", flag, data, err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
