package afero

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"testing"
)

func TestRegexpDirectoryReads(t *testing.T) {
	for _, backend := range []string{"memory", "os"} {
		for _, count := range []int{1, 2, 0, -1} {
			for _, filter := range []string{"matching", "none", "nil"} {
				t.Run(fmt.Sprintf("%s/%d/%s", backend, count, filter), func(t *testing.T) {
					defer func() {
						if recovered := recover(); recovered != nil {
							t.Fatalf("directory read panicked: %v", recovered)
						}
					}()
					var source Fs = NewMemMapFs()
					if backend == "os" {
						source = NewBasePathFs(NewOsFs(), t.TempDir())
					}
					for _, name := range []string{"a.txt", "b.yaml", "c.txt", "d.yaml"} {
						if err := WriteFile(source, name, nil, 0600); err != nil {
							t.Fatal(err)
						}
					}
					if err := source.Mkdir("subdir", 0700); err != nil {
						t.Fatal(err)
					}
					re := regexp.MustCompile(`\.yaml$`)
					want := []string{"b.yaml", "d.yaml", "subdir"}
					if filter == "none" {
						re = regexp.MustCompile(`never-match`)
						want = []string{"subdir"}
					} else if filter == "nil" {
						re = nil
						want = []string{"a.txt", "b.yaml", "c.txt", "d.yaml", "subdir"}
					}
					file, err := NewRegexpFs(source, re).Open(".")
					if err != nil {
						t.Fatal(err)
					}
					defer file.Close()
					var got []string
					for call := 0; ; call++ {
						if call > 6 {
							t.Fatal("directory did not reach EOF")
						}
						var names []string
						// Both APIs must advance the same underlying cursor.
						if call%2 == 0 {
							entries, readErr := file.Readdir(count)
							err = readErr
							for _, entry := range entries {
								names = append(names, entry.Name())
							}
						} else {
							names, err = file.Readdirnames(count)
						}
						if count > 0 && (len(names) > count || (len(names) == 0 && err == nil)) {
							t.Fatalf("read(%d) = %v, %v", count, names, err)
						}
						if err != nil && !errors.Is(err, io.EOF) {
							t.Fatal(err)
						}
						got = append(got, names...)
						if count <= 0 || err != nil {
							break
						}
					}
					sort.Strings(got)
					if fmt.Sprint(got) != fmt.Sprint(want) {
						t.Fatalf("directory entries = %v, want %v", got, want)
					}
				})
			}
		}
	}
}

type regexpReadErrorFile struct {
	File
	err error
}

func (f regexpReadErrorFile) Readdir(n int) ([]os.FileInfo, error) {
	entries, err := f.File.Readdir(n)
	if err != nil {
		return entries, err
	}
	return entries, f.err
}

func TestRegexpDirectoryReadError(t *testing.T) {
	for _, readErr := range []error{io.EOF, errors.New("directory read failed")} {
		for _, namesOnly := range []bool{false, true} {
			t.Run(fmt.Sprintf("%v/names=%v", readErr, namesOnly), func(t *testing.T) {
				source := NewMemMapFs()
				if err := WriteFile(source, "a.yaml", nil, 0600); err != nil {
					t.Fatal(err)
				}
				file, err := source.Open(".")
				if err != nil {
					t.Fatal(err)
				}
				defer file.Close()
				filtered := &RegexpFile{f: regexpReadErrorFile{File: file, err: readErr}, re: regexp.MustCompile(`\.yaml$`)}
				var got []string
				if namesOnly {
					got, err = filtered.Readdirnames(2)
				} else {
					var entries []os.FileInfo
					entries, err = filtered.Readdir(2)
					for _, entry := range entries {
						got = append(got, entry.Name())
					}
				}
				if len(got) != 1 || got[0] != "a.yaml" || !errors.Is(err, readErr) {
					t.Fatalf("directory read = %v, %v; want [a.yaml], %v", got, err, readErr)
				}
			})
		}
	}
}
