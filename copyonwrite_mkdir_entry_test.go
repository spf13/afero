package afero

import (
	"errors"
	"os"
	"syscall"
	"testing"
)

func TestCopyOnWriteMkdirVisibleEntry(t *testing.T) {
	const name = "entry"
	states := []string{"missing", "file", "directory"}
	for _, operation := range []string{"Mkdir", "MkdirAll"} {
		t.Run(operation, func(t *testing.T) {
			for _, baseState := range states {
				for _, layerState := range states {
					t.Run("base-"+baseState+"_layer-"+layerState, func(t *testing.T) {
						base, layer := NewMemMapFs(), NewMemMapFs()
						setup := func(fs Fs, state, content string) {
							t.Helper()
							var err error
							if state == "file" {
								err = WriteFile(fs, name, []byte(content), 0o644)
							} else if state == "directory" {
								err = fs.Mkdir(name, 0o755)
							}
							if err != nil {
								t.Fatal(err)
							}
						}
						setup(base, baseState, "base")
						setup(layer, layerState, "layer")
						union := NewCopyOnWriteFs(NewReadOnlyFs(base), layer)
						visible := layerState
						if visible == "missing" {
							visible = baseState
						}
						var err error
						if operation == "Mkdir" {
							err = union.Mkdir(name, 0o700)
						} else {
							err = union.MkdirAll(name, 0o700)
						}
						if operation == "Mkdir" && visible != "missing" {
							if !os.IsExist(err) || !errors.Is(err, os.ErrExist) {
								t.Errorf("%s returned %v; want ErrExist", operation, err)
							}
						} else if visible == "file" {
							if !errors.Is(err, syscall.ENOTDIR) {
								t.Errorf("%s returned %v; want ENOTDIR", operation, err)
							}
						} else if err != nil {
							t.Errorf("%s returned %v; want success", operation, err)
						}
						assertState := func(fs Fs, want, content string) {
							t.Helper()
							info, err := fs.Stat(name)
							if want == "missing" {
								if !os.IsNotExist(err) {
									t.Errorf("entry unexpectedly present: %v, %v", info, err)
								}
								return
							}
							if err != nil {
								t.Fatal(err)
							}
							if info.IsDir() != (want == "directory") {
								t.Errorf("entry type changed: %v; want %s", info.Mode(), want)
							}
							if want == "file" {
								data, err := ReadFile(fs, name)
								if err != nil || string(data) != content {
									t.Errorf("file content = %q, %v; want %q", data, err, content)
								}
							}
						}
						assertState(base, baseState, "base")
						wantLayer := layerState
						if visible == "missing" {
							wantLayer = "directory"
							visible = "directory"
						}
						assertState(layer, wantLayer, "layer")
						content := "layer"
						if layerState == "missing" {
							content = "base"
						}
						assertState(union, visible, content)
					})
				}
			}
		})
	}
}

func TestCopyOnWriteMkdirBaseParent(t *testing.T) {
	base, layer := NewMemMapFs(), NewMemMapFs()
	if err := base.Mkdir("parent", 0o755); err != nil {
		t.Fatal(err)
	}
	union := NewCopyOnWriteFs(NewReadOnlyFs(base), layer)
	if err := union.Mkdir("parent/child", 0o700); err != nil {
		t.Fatal(err)
	}
	info, err := layer.Stat("parent/child")
	if err != nil || !info.IsDir() {
		t.Fatalf("overlay child = %v, %v", info, err)
	}
	if _, err := base.Stat("parent/child"); !os.IsNotExist(err) {
		t.Errorf("base was changed: %v", err)
	}
}

func TestCopyOnWriteMkdirOsLayer(t *testing.T) {
	layer := NewBasePathFs(NewOsFs(), t.TempDir())
	if err := WriteFile(layer, "entry", []byte("layer"), 0o644); err != nil {
		t.Fatal(err)
	}
	union := NewCopyOnWriteFs(NewReadOnlyFs(NewMemMapFs()), layer)
	if err := union.MkdirAll("entry", 0o700); !errors.Is(err, syscall.ENOTDIR) {
		t.Errorf("MkdirAll on an existing file = %v; want ENOTDIR", err)
	}
	data, err := ReadFile(layer, "entry")
	if err != nil || string(data) != "layer" {
		t.Errorf("overlay file changed: %q, %v", data, err)
	}
}

type mkdirStatErrorFS struct {
	Fs
	statErr error
	mkdirs  int
}

func (f *mkdirStatErrorFS) Stat(name string) (os.FileInfo, error) {
	if f.statErr != nil {
		return nil, f.statErr
	}
	return f.Fs.Stat(name)
}

func (f *mkdirStatErrorFS) MkdirAll(name string, perm os.FileMode) error {
	f.mkdirs++
	return f.Fs.MkdirAll(name, perm)
}

func TestCopyOnWriteMkdirStatError(t *testing.T) {
	for _, failingLayer := range []bool{false, true} {
		for _, operation := range []string{"Mkdir", "MkdirAll"} {
			base := &mkdirStatErrorFS{Fs: NewMemMapFs()}
			layer := &mkdirStatErrorFS{Fs: NewMemMapFs()}
			want := &os.PathError{Op: "stat", Path: "entry", Err: os.ErrPermission}
			if failingLayer {
				layer.statErr = want
			} else {
				base.statErr = want
			}
			union := NewCopyOnWriteFs(base, layer)
			var err error
			if operation == "Mkdir" {
				err = union.Mkdir("entry", 0o700)
			} else {
				err = union.MkdirAll("entry", 0o700)
			}
			if err != want || layer.mkdirs != 0 {
				t.Errorf("%s (layer error=%t): err=%v overlay mkdirs=%d; want original error and no mutation", operation, failingLayer, err, layer.mkdirs)
			}
		}
	}
}
