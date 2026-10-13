package afero

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
)

type readDirPathFS struct {
	Fs
	opens []string
}

func (f *readDirPathFS) Open(name string) (File, error) {
	f.opens = append(f.opens, name)
	return f.Fs.Open(name)
}

func TestIOFSReadDirInvalidPath(t *testing.T) {
	for _, name := range []string{"", "/", "/dir", "dir/", "dir//sub", "dir/./sub", "dir/../dir", "../dir", "\xff"} {
		t.Run(name, func(t *testing.T) {
			backend := &readDirPathFS{Fs: NewMemMapFs()}
			if err := backend.MkdirAll("dir/sub", 0o755); err != nil {
				t.Fatal(err)
			}
			entries, err := NewIOFS(backend).ReadDir(name)
			var pathErr *fs.PathError
			if !errors.As(err, &pathErr) || !errors.Is(err, fs.ErrInvalid) {
				t.Errorf("ReadDir(%q) = %v; want PathError wrapping ErrInvalid", name, err)
			} else if pathErr.Op != "readdir" || pathErr.Path != name {
				t.Errorf("unexpected PathError: %+v", pathErr)
			}
			if entries != nil {
				t.Errorf("ReadDir(%q) returned entries: %v", name, entries)
			}
			if len(backend.opens) != 0 {
				t.Errorf("invalid name reached the backend: %q", backend.opens)
			}
		})
	}
}

func TestIOFSReadDirValidPath(t *testing.T) {
	// MapFS keeps virtual path semantics independent of the host OS. In io/fs,
	// backslash and colon are valid characters, not path separators.
	files := fstest.MapFS{
		"dir/z":           {Data: []byte("z")},
		"dir/a":           {Data: []byte("a")},
		`back\slash/file`: {Data: []byte("b")},
		"colon:name/file": {Data: []byte("c")},
	}
	for _, tc := range []struct {
		name string
		want []string
	}{
		{name: ".", want: []string{`back\slash`, "colon:name", "dir"}},
		{name: "dir", want: []string{"a", "z"}},
		{name: `back\slash`, want: []string{"file"}},
		{name: "colon:name", want: []string{"file"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			backend := &readDirPathFS{Fs: FromIOFS{FS: files}}
			entries, err := NewIOFS(backend).ReadDir(tc.name)
			if err != nil {
				t.Fatal(err)
			}
			if len(backend.opens) != 1 || backend.opens[0] != tc.name {
				t.Errorf("backend opens = %q; want exactly %q", backend.opens, tc.name)
			}
			if len(entries) != len(tc.want) {
				t.Fatalf("ReadDir(%q) returned %d entries; want %d", tc.name, len(entries), len(tc.want))
			}
			for i, entry := range entries {
				if entry.Name() != tc.want[i] {
					t.Errorf("entry %d = %q; want %q", i, entry.Name(), tc.want[i])
				}
			}
		})
	}
}
