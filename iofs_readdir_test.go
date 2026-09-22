//go:build go1.16
// +build go1.16

package afero

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"testing"
)

type readdirResultFs struct {
	Fs
	file File
}

func (f readdirResultFs) Open(string) (File, error) { return f.file, nil }

type readdirResultFile struct {
	File
	infos  []os.FileInfo
	err    error
	count  int
	closed bool
}

func (f *readdirResultFile) Readdir(n int) ([]os.FileInfo, error) {
	f.count = n
	return f.infos, f.err
}

func (f *readdirResultFile) Close() error {
	f.closed = true
	return nil
}

type nativeReaddirResultFile struct {
	*readdirResultFile
	entries []fs.DirEntry
}

func (f *nativeReaddirResultFile) ReadDir(n int) ([]fs.DirEntry, error) {
	f.count = n
	return f.entries, f.err
}

func readdirResultInfos(t *testing.T) []os.FileInfo {
	t.Helper()
	source := NewMemMapFs()
	if err := WriteFile(source, "z.txt", []byte("contents"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := source.Mkdir("a", 0o700); err != nil {
		t.Fatal(err)
	}
	var infos []os.FileInfo
	for _, name := range []string{"z.txt", "a"} {
		info, err := source.Stat(name)
		if err != nil {
			t.Fatal(err)
		}
		infos = append(infos, info)
	}
	return infos
}

func assertReaddirResult(t *testing.T, entries []fs.DirEntry, infos []os.FileInfo) {
	t.Helper()
	if len(entries) != len(infos) {
		t.Fatalf("got %d entries, want %d", len(entries), len(infos))
	}
	for i, entry := range entries {
		if entry.Name() != infos[i].Name() || entry.Type() != infos[i].Mode().Type() ||
			entry.IsDir() != infos[i].IsDir() {
			t.Errorf("entry %d = %v, want %v", i, entry, infos[i])
		}
		info, err := entry.Info()
		if err != nil || info != infos[i] {
			t.Errorf("entry %d Info() = %v, %v, want original FileInfo %v", i, info, err, infos[i])
		}
	}
}

func TestIOFSReadDirFileResults(t *testing.T) {
	infos := readdirResultInfos(t)
	readErr := errors.New("directory read failed")
	for _, tt := range []struct {
		name  string
		n     int
		infos []os.FileInfo
		err   error
	}{
		{name: "partial EOF", n: 3, infos: infos, err: io.EOF},
		{name: "partial error", n: 3, infos: infos, err: readErr},
		{name: "partial read all error", n: -1, infos: infos, err: readErr},
		{name: "partial zero count error", n: 0, infos: infos, err: readErr},
		{name: "nil EOF", n: 1, err: io.EOF},
		{name: "empty EOF", n: 1, infos: []os.FileInfo{}, err: io.EOF},
		{name: "nil error result", n: -1, err: readErr},
		{name: "empty error result", n: -1, infos: []os.FileInfo{}, err: readErr},
		{name: "success", n: 2, infos: infos},
		{name: "read all success", n: -1, infos: infos},
		{name: "empty success", n: -1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			file := &readdirResultFile{infos: tt.infos, err: tt.err}
			opened, err := NewIOFS(readdirResultFs{file: file}).Open(".")
			if err != nil {
				t.Fatal(err)
			}
			defer opened.Close()
			entries, err := opened.(fs.ReadDirFile).ReadDir(tt.n)
			if err != tt.err {
				t.Errorf("ReadDir error = %v, want original error %v", err, tt.err)
			}
			if file.count != tt.n {
				t.Errorf("backend count = %d, want %d", file.count, tt.n)
			}
			assertReaddirResult(t, entries, tt.infos)
			if len(tt.infos) == 0 && (entries == nil) != (tt.err != nil) {
				t.Errorf("ReadDir returned nil slice = %t, want %t", entries == nil, tt.err != nil)
			}
		})
	}
}

func TestIOFSReadDirResults(t *testing.T) {
	infos := readdirResultInfos(t)
	readErr := errors.New("directory read failed")
	pathErr := &fs.PathError{Op: "backend readdir", Path: "backend path", Err: readErr}
	for _, native := range []bool{false, true} {
		backend := "fallback"
		if native {
			backend = "native"
		}
		t.Run(backend, func(t *testing.T) {
			for _, tt := range []struct {
				name  string
				infos []os.FileInfo
				err   error
			}{
				{name: "partial error", infos: infos, err: readErr},
				{name: "partial path error", infos: infos, err: pathErr},
				{name: "nil error result", err: readErr},
				{name: "empty error result", infos: []os.FileInfo{}, err: readErr},
				{name: "success", infos: infos},
				{name: "nil success"},
				{name: "empty success", infos: []os.FileInfo{}},
			} {
				t.Run(tt.name, func(t *testing.T) {
					file := &readdirResultFile{err: tt.err}
					if tt.infos != nil {
						file.infos = make([]os.FileInfo, len(tt.infos))
						copy(file.infos, tt.infos)
					}
					var source File = file
					var nativeEntries []fs.DirEntry
					if native {
						if tt.infos != nil {
							nativeEntries = make([]fs.DirEntry, len(tt.infos))
						}
						for i, info := range tt.infos {
							nativeEntries[i] = fs.FileInfoToDirEntry(info)
						}
						nativeFile := &nativeReaddirResultFile{
							readdirResultFile: file,
						}
						if nativeEntries != nil {
							nativeFile.entries = make([]fs.DirEntry, len(nativeEntries))
							copy(nativeFile.entries, nativeEntries)
						}
						source = nativeFile
					}
					entries, err := NewIOFS(readdirResultFs{file: source}).ReadDir("dir")
					if !errors.Is(err, tt.err) {
						t.Errorf("ReadDir error = %v, want %v", err, tt.err)
					}
					if tt.err == pathErr {
						if err != pathErr {
							t.Errorf("ReadDir replaced existing PathError: %v", err)
						}
					} else if tt.err != nil {
						var gotErr *fs.PathError
						if !errors.As(err, &gotErr) || gotErr.Op != "readdir" || gotErr.Path != "dir" {
							t.Errorf("ReadDir did not wrap error with operation and path: %v", err)
						}
					}
					if file.count != -1 || !file.closed {
						t.Errorf(
							"backend count = %d, closed = %t; want -1, true",
							file.count,
							file.closed,
						)
					}
					var sorted []os.FileInfo
					if len(tt.infos) != 0 {
						sorted = []os.FileInfo{tt.infos[1], tt.infos[0]}
					}
					assertReaddirResult(t, entries, sorted)
					if tt.err != nil && len(tt.infos) == 0 && entries != nil {
						t.Error("empty error result should return a nil slice")
					}
					if tt.err == nil && len(tt.infos) == 0 {
						wantNil := native && tt.infos == nil
						if (entries == nil) != wantNil {
							t.Errorf(
								"ReadDir returned nil slice = %t, want %t",
								entries == nil,
								wantNil,
							)
						}
					}
					if native && len(entries) != 0 &&
						(entries[0] != nativeEntries[1] || entries[1] != nativeEntries[0]) {
						t.Error("ReadDir replaced native DirEntry values")
					}
				})
			}
		})
	}
}
