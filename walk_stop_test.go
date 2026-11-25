package afero

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWalkStopSentinels(t *testing.T) {
	for _, backend := range []string{"memory", "os"} {
		t.Run(backend, func(t *testing.T) {
			source := NewMemMapFs()
			if backend == "os" {
				source = NewBasePathFs(NewOsFs(), t.TempDir())
			}
			if err := source.MkdirAll(filepath.Join("tree", "sub"), 0o755); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{filepath.Join("tree", "a.txt"), filepath.Join("tree", "sub", "b.txt")} {
				if err := WriteFile(source, name, []byte("data"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			for _, root := range []string{"tree", filepath.Join("tree", "a.txt"), "missing"} {
				t.Run(root, func(t *testing.T) {
					var visited []string
					err := Walk(source, root, func(path string, info os.FileInfo, err error) error {
						visited = append(visited, path)
						return filepath.SkipAll
					})
					if err != nil || !reflect.DeepEqual(visited, []string{root}) {
						t.Fatalf("Walk = %v, visited %v; want nil and [%s]", err, visited, root)
					}
				})
			}
			var visited []string
			err := (Afero{Fs: source}).Walk(
				"tree",
				func(path string, info os.FileInfo, err error) error {
					visited = append(visited, path)
					if path == filepath.Join("tree", "a.txt") {
						return filepath.SkipAll
					}
					return err
				},
			)
			want := []string{"tree", filepath.Join("tree", "a.txt")}
			if err != nil || !reflect.DeepEqual(visited, want) {
				t.Fatalf("Walk = %v, visited %v; want nil and %v", err, visited, want)
			}
		})
	}
}

func TestWalkRootSkipDirAndCallbackError(t *testing.T) {
	source := NewMemMapFs()
	if err := WriteFile(source, "data", nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Walk(source, "data", func(string, os.FileInfo, error) error {
		return filepath.SkipDir
	}); err != nil {
		t.Fatalf("Walk returning SkipDir at root = %v, want nil", err)
	}
	want := errors.New("consumer failed")
	if err := Walk(source, "data", func(string, os.FileInfo, error) error {
		return want
	}); !errors.Is(err, want) {
		t.Fatalf("Walk = %v, want callback error %v", err, want)
	}
}
