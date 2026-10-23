package afero_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/spf13/afero"
)

func TestMemMapFsConcurrentRenameSameSourceFile(t *testing.T) {
	testMemMapFsConcurrentRenameSameSource(t, map[string]string{
		"": "file payload",
	}, nil)
}

func TestMemMapFsConcurrentRenameSameSourceDirectory(t *testing.T) {
	testMemMapFsConcurrentRenameSameSource(t, map[string]string{
		"root.txt":                                  "root payload",
		filepath.Join("nested", "child.txt"):        "nested payload",
		filepath.Join("nested", "deep", "leaf.txt"): "deep payload",
	}, map[string][]string{
		"":                              {"empty", "nested", "root.txt"},
		"empty":                         {},
		"nested":                        {"child.txt", "deep"},
		filepath.Join("nested", "deep"): {"leaf.txt"},
	})
}

func testMemMapFsConcurrentRenameSameSource(
	t *testing.T,
	files map[string]string,
	dirs map[string][]string,
) {
	t.Helper()
	// Exercise concurrent execution even when the test runner defaults to one CPU.
	if procs := runtime.GOMAXPROCS(0); procs < 4 {
		runtime.GOMAXPROCS(4)
		defer runtime.GOMAXPROCS(procs)
	}

	const rounds = 64
	const contenders = 32
	for round := 0; round < rounds; round++ {
		t.Run(fmt.Sprintf("round_%02d", round), func(t *testing.T) {
			fs := afero.NewMemMapFs()
			root := string(filepath.Separator)
			sourceParent := filepath.Join(root, "source")
			source := filepath.Join(sourceParent, "item")
			destinationRoot := filepath.Join(root, "destinations")
			if err := afero.WriteFile(
				fs,
				filepath.Join(sourceParent, "keep.txt"),
				[]byte("untouched"),
				0o644,
			); err != nil {
				t.Fatal(err)
			}
			for dir := range dirs {
				if err := fs.MkdirAll(filepath.Join(source, dir), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			for file, payload := range files {
				if err := afero.WriteFile(
					fs,
					filepath.Join(source, file),
					[]byte(payload),
					0o644,
				); err != nil {
					t.Fatal(err)
				}
			}

			destinations := make([]string, contenders)
			parentNames := make([]string, contenders)
			for i := range destinations {
				parentNames[i] = fmt.Sprintf("destination_%02d", i)
				parent := filepath.Join(destinationRoot, parentNames[i])
				if err := fs.MkdirAll(parent, 0o755); err != nil {
					t.Fatal(err)
				}
				destinations[i] = filepath.Join(parent, "item")
			}

			type result struct {
				destination string
				err         error
			}
			results := make(chan result, contenders)
			start := make(chan struct{})
			var ready sync.WaitGroup
			ready.Add(contenders)
			for _, destination := range destinations {
				go func() {
					ready.Done()
					<-start
					results <- result{destination, fs.Rename(source, destination)}
				}()
			}
			ready.Wait()
			close(start)

			deadline := time.NewTimer(10 * time.Second)
			defer deadline.Stop()
			winner := ""
			wins := 0
			for range contenders {
				select {
				case r := <-results:
					if r.err == nil {
						winner = r.destination
						wins++
						continue
					}
					if !os.IsNotExist(r.err) {
						t.Errorf(
							"Rename(%q, %q) = %v; want a missing-source error",
							source,
							r.destination,
							r.err,
						)
					}
					pathErr, ok := r.err.(*os.PathError)
					if !ok || pathErr.Op != "rename" || pathErr.Path != source {
						t.Errorf(
							"Rename(%q, %q) = %#v; want a rename PathError for the source",
							source,
							r.destination,
							r.err,
						)
					}
				case <-deadline.C:
					t.Fatal("concurrent Rename calls did not finish")
				}
			}
			if wins != 1 {
				t.Fatalf("successful Rename calls = %d; want exactly one", wins)
			}

			assertMemMapRenameMissing(t, fs, source)
			for file := range files {
				assertMemMapRenameMissing(t, fs, filepath.Join(source, file))
			}
			for dir := range dirs {
				assertMemMapRenameMissing(t, fs, filepath.Join(source, dir))
			}
			info, err := fs.Stat(winner)
			if err != nil {
				t.Fatal(err)
			}
			if info.Name() != "item" || info.IsDir() != (dirs != nil) {
				t.Errorf(
					"winning destination: name = %q, directory = %t",
					info.Name(),
					info.IsDir(),
				)
			}
			for file, payload := range files {
				assertMemMapRenamePayload(t, fs, filepath.Join(winner, file), payload)
			}
			for dir, entries := range dirs {
				assertMemMapRenameEntries(t, fs, filepath.Join(winner, dir), entries...)
			}
			for _, destination := range destinations {
				parent := filepath.Dir(destination)
				if destination == winner {
					assertMemMapRenameEntries(t, fs, parent, "item")
					continue
				}
				assertMemMapRenameEntries(t, fs, parent)
				assertMemMapRenameMissing(t, fs, destination)
				for file := range files {
					assertMemMapRenameMissing(t, fs, filepath.Join(destination, file))
				}
				for dir := range dirs {
					assertMemMapRenameMissing(t, fs, filepath.Join(destination, dir))
				}
			}
			assertMemMapRenameEntries(t, fs, sourceParent, "keep.txt")
			assertMemMapRenamePayload(t, fs, filepath.Join(sourceParent, "keep.txt"), "untouched")
			assertMemMapRenameEntries(t, fs, destinationRoot, parentNames...)
			assertMemMapRenameEntries(t, fs, root, "destinations", "source")
		})
	}
}

func assertMemMapRenameMissing(t *testing.T, fs afero.Fs, name string) {
	t.Helper()
	if _, err := fs.Stat(name); !os.IsNotExist(err) {
		t.Errorf("Stat(%q) = %v; want a missing-path error", name, err)
	}
}

func assertMemMapRenamePayload(t *testing.T, fs afero.Fs, name, want string) {
	t.Helper()
	got, err := afero.ReadFile(fs, name)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", name, err)
	}
	if string(got) != want {
		t.Errorf("ReadFile(%q) = %q; want %q", name, got, want)
	}
}

func assertMemMapRenameEntries(t *testing.T, fs afero.Fs, dir string, want ...string) {
	t.Helper()
	entries, err := afero.ReadDir(fs, dir)
	if err != nil {
		t.Fatalf("ReadDir(%q): %v", dir, err)
	}
	if len(entries) != len(want) {
		t.Fatalf("ReadDir(%q) has %d entries; want %v", dir, len(entries), want)
	}
	for i, entry := range entries {
		if entry.Name() != want[i] {
			t.Errorf("ReadDir(%q) entry %d = %q; want %q", dir, i, entry.Name(), want[i])
		}
		info, err := fs.Stat(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("listed entry %q in %q cannot be statted: %v", entry.Name(), dir, err)
		}
		if entry.Mode() != info.Mode() || entry.Size() != info.Size() {
			t.Errorf("listed entry %q in %q disagrees with Stat", entry.Name(), dir)
		}
	}
}
