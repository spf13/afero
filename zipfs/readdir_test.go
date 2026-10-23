package zipfs

import (
	"archive/zip"
	"bytes"
	"io"
	"reflect"
	"sort"
	"syscall"
	"testing"

	"github.com/spf13/afero"
)

func directoryReadFS(t *testing.T, names ...string) afero.Fs {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, name := range names {
		if _, err := w.Create(name); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	return New(r)
}

func directoryReadNames(f afero.File, count int, useInfo bool) ([]string, error) {
	if !useInfo {
		return f.Readdirnames(count)
	}
	entries, err := f.Readdir(count)
	if entries == nil {
		return nil, err
	}
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names, err
}

func TestDirectoryReadClosed(t *testing.T) {
	fs := directoryReadFS(t, "a", "b")
	f, err := fs.Open("/")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Readdirnames(1); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	for _, useInfo := range []bool{false, true} {
		if _, err := directoryReadNames(f, 1, useInfo); err != afero.ErrFileClosed {
			t.Fatalf("closed read: %v", err)
		}
	}
}

func TestDirectoryReadCursor(t *testing.T) {
	for _, path := range []string{"/", "/implicit", "/explicit"} {
		for _, infoFirst := range []bool{false, true} {
			for label, counts := range map[string][]int{
				"single":             {1, 1, 1},
				"partial":            {2, 2},
				"zero_remaining":     {1, 0},
				"negative_remaining": {1, -1},
				"large_count":        {int(^uint(0) >> 1)},
			} {
				t.Run(path+"/"+label+"/infoFirst="+map[bool]string{false: "false", true: "true"}[infoFirst], func(t *testing.T) {
					fs := directoryReadFS(t, "a", "implicit/a", "implicit/b", "implicit/c", "explicit/", "explicit/a", "explicit/b", "explicit/c")
					f, err := fs.Open(path)
					if err != nil {
						t.Fatal(err)
					}
					defer f.Close()
					var got []string
					for i, count := range counts {
						batch, err := directoryReadNames(f, count, infoFirst != (i%2 != 0))
						if err != nil {
							t.Fatalf("read %d: %v", i, err)
						}
						remaining := 3 - len(got)
						want := remaining
						if count > 0 && count < want {
							want = count
						}
						if len(batch) != want {
							t.Fatalf("read %d: got %v, want %d entries", i, batch, want)
						}
						got = append(got, batch...)
					}
					sort.Strings(got)
					want := []string{"a", "b", "c"}
					if path == "/" {
						want = []string{"a", "explicit", "implicit"}
					}
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("entries=%v, want %v", got, want)
					}
					for i := 0; i < 2; i++ {
						batch, err := directoryReadNames(f, 1, i == 0)
						if len(batch) != 0 || err != io.EOF {
							t.Fatalf("exhausted: %v, %v", batch, err)
						}
					}
					for _, count := range []int{0, -1} {
						batch, err := directoryReadNames(f, count, count == 0)
						if batch == nil || len(batch) != 0 || err != nil {
							t.Fatalf("unbounded exhausted: %v, %v", batch, err)
						}
					}
				})
			}
		}
	}
}

func TestDirectoryReadEmptyAndIndependent(t *testing.T) {
	fs := directoryReadFS(t, "empty/", "a", "b")
	for _, path := range []string{"empty", "/"} {
		t.Run(path, func(t *testing.T) {
			f, err := fs.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			g, err := fs.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer g.Close()
			first, firstErr := f.Readdirnames(1)
			other, otherErr := g.Readdirnames(-1)
			if path == "empty" {
				if len(first) != 0 || firstErr != io.EOF || other == nil || len(other) != 0 || otherErr != nil {
					t.Fatalf("empty: %v/%v; %v/%v", first, firstErr, other, otherErr)
				}
			} else {
				if len(first) != 1 || firstErr != nil || len(other) != 3 || otherErr != nil {
					t.Fatalf("independent: %v/%v; %v/%v", first, firstErr, other, otherErr)
				}
			}
		})
	}
	f, err := fs.Open("a")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.Readdir(1); err != syscall.ENOTDIR {
		t.Fatalf("file Readdir: %v", err)
	}
	if _, err := f.Readdirnames(1); err != syscall.ENOTDIR {
		t.Fatalf("file Readdirnames: %v", err)
	}
}
