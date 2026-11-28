package afero_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
)

func TestCacheOnReadRemove(t *testing.T) {
	for _, duration := range []time.Duration{0, time.Hour} {
		for _, directory := range []bool{false, true} {
			for _, cached := range []bool{false, true} {
				for _, base := range []afero.Fs{afero.NewMemMapFs(), afero.NewOsFs()} {
					name := base.Name() + "/" + duration.String()
					if directory {
						name += "/directory"
					} else {
						name += "/file"
					}
					if cached {
						name += "/cached"
					} else {
						name += "/uncached"
					}
					t.Run(name, func(t *testing.T) {
						layer := afero.NewMemMapFs()
						root := t.TempDir()
						path := filepath.Join(root, "entry")
						create := func(fs afero.Fs) {
							t.Helper()
							if err := fs.MkdirAll(root, 0o700); err != nil {
								t.Fatal(err)
							}
							if directory {
								if err := fs.Mkdir(path, 0o700); err != nil {
									t.Fatal(err)
								}
							} else if err := afero.WriteFile(fs, path, []byte("payload"), 0o600); err != nil {
								t.Fatal(err)
							}
						}
						create(base)
						if cached {
							create(layer)
						}
						fs := afero.NewCacheOnReadFs(base, layer, duration)
						if err := fs.Remove(path); err != nil {
							t.Errorf("Remove returned %v for an existing entry", err)
						}
						for _, backing := range []afero.Fs{base, layer} {
							if _, err := backing.Stat(path); !os.IsNotExist(err) {
								t.Errorf(
									"%s Stat after Remove = %v, want not-exist",
									backing.Name(),
									err,
								)
							}
						}
						if err := fs.Remove(path); !os.IsNotExist(err) {
							t.Errorf("second Remove = %v, want not-exist", err)
						}
					})
				}
			}
		}
	}
}
