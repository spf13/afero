package afero_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

type competingCreateFS struct {
	afero.Fs
	path string
	done bool
}

func (f *competingCreateFS) createCompetingFile(name string) error {
	if name != f.path || f.done {
		return nil
	}
	f.done = true
	return afero.WriteFile(f.Fs, name, []byte("other writer's data"), 0o600)
}

func (f *competingCreateFS) Create(name string) (afero.File, error) {
	if err := f.createCompetingFile(name); err != nil {
		return nil, err
	}
	return f.Fs.Create(name)
}

func (f *competingCreateFS) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if err := f.createCompetingFile(name); err != nil {
		return nil, err
	}
	return f.Fs.OpenFile(name, flag, perm)
}

func TestSafeWriteReaderDoesNotOverwriteCompetingCreate(t *testing.T) {
	for _, fs := range []afero.Fs{afero.NewMemMapFs(), afero.NewOsFs()} {
		t.Run(fs.Name(), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "output")
			interleaved := &competingCreateFS{Fs: fs, path: path}
			err := afero.SafeWriteReader(interleaved, path, strings.NewReader("replacement"))
			if err == nil {
				t.Error("SafeWriteReader reported success after another writer created the file")
			}
			data, err := afero.ReadFile(fs, path)
			if err != nil || string(data) != "other writer's data" {
				t.Errorf("competing file was overwritten: (%q, %v)", data, err)
			}
		})
	}
}

func TestSafeWriteReaderCreatesNewFile(t *testing.T) {
	for _, fs := range []afero.Fs{afero.NewMemMapFs(), afero.NewOsFs()} {
		t.Run(fs.Name(), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "nested", "output")
			if err := afero.SafeWriteReader(fs, path, strings.NewReader("payload")); err != nil {
				t.Fatal(err)
			}
			if err := afero.SafeWriteReader(
				fs,
				path,
				strings.NewReader("replacement"),
			); err == nil {
				t.Error("SafeWriteReader overwrote an existing file")
			}
			data, err := afero.ReadFile(fs, path)
			if err != nil || string(data) != "payload" {
				t.Errorf("new file = (%q, %v)", data, err)
			}
		})
	}
}
