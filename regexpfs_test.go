package afero

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestRegexpFsOpenFile(t *testing.T) {
	for _, backend := range []string{"MemMapFs", "OsFs"} {
		t.Run(backend, func(t *testing.T) {
			for _, tt := range []struct {
				name       string
				path       string
				flag       int
				existing   bool
				directory  bool
				unfiltered bool
				wantErr    error
				wantData   string
			}{
				{name: "create matching file", path: "config.yaml", flag: os.O_CREATE | os.O_WRONLY},
				{name: "create exclusive matching file", path: "config.yaml", flag: os.O_CREATE | os.O_EXCL | os.O_WRONLY},
				{name: "missing without create", path: "config.yaml", flag: os.O_RDONLY, wantErr: os.ErrNotExist},
				{name: "reject missing nonmatching file", path: "config.json", flag: os.O_CREATE | os.O_WRONLY, wantErr: os.ErrNotExist},
				{name: "preserve rejected file", path: "config.json", flag: os.O_CREATE | os.O_TRUNC | os.O_WRONLY, existing: true, wantErr: os.ErrNotExist, wantData: "original"},
				{name: "preserve existing matching file", path: "config.yaml", flag: os.O_CREATE | os.O_WRONLY, existing: true, wantData: "original"},
				{name: "exclusive existing file", path: "config.yaml", flag: os.O_CREATE | os.O_EXCL | os.O_WRONLY, existing: true, wantErr: os.ErrExist, wantData: "original"},
				{name: "truncate matching file", path: "config.yaml", flag: os.O_CREATE | os.O_TRUNC | os.O_WRONLY, existing: true},
				{name: "open nonmatching directory", path: "directory", flag: os.O_RDONLY, directory: true},
				{name: "nil regexp", path: "config.json", flag: os.O_CREATE | os.O_WRONLY, unfiltered: true},
			} {
				t.Run(tt.name, func(t *testing.T) {
					source := NewMemMapFs()
					if backend == "OsFs" {
						source = NewBasePathFs(NewOsFs(), t.TempDir())
					}
					if tt.existing {
						if err := WriteFile(source, tt.path, []byte("original"), 0o600); err != nil {
							t.Fatal(err)
						}
					}
					if tt.directory {
						if err := source.Mkdir(tt.path, 0o700); err != nil {
							t.Fatal(err)
						}
					}
					filter := regexp.MustCompile(`\.yaml$`)
					if tt.unfiltered {
						filter = nil
					}
					file, err := NewRegexpFs(source, filter).OpenFile(tt.path, tt.flag, 0o600)
					if file != nil {
						if closeErr := file.Close(); closeErr != nil {
							t.Error(closeErr)
						}
					}
					if !errors.Is(err, tt.wantErr) {
						t.Fatalf("OpenFile error = %v, want %v", err, tt.wantErr)
					}
					if tt.directory {
						return
					}
					if tt.wantErr != nil && !tt.existing {
						if _, err := source.Stat(tt.path); !os.IsNotExist(err) {
							t.Fatalf("failed OpenFile created a file: Stat error = %v", err)
						}
						return
					}
					data, err := ReadFile(source, tt.path)
					if err != nil {
						t.Fatal(err)
					}
					if string(data) != tt.wantData {
						t.Errorf("file contents = %q, want %q", data, tt.wantData)
					}
				})
			}
		})
	}
}

func TestRegexpFsOpenFileMissingParent(t *testing.T) {
	source := NewOsFs()
	filter := NewRegexpFs(source, regexp.MustCompile(`\.yaml$`))
	path := filepath.Join(t.TempDir(), "missing", "config.yaml")
	file, err := filter.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if file != nil {
		file.Close()
	}
	if !os.IsNotExist(err) {
		t.Fatalf("OpenFile error = %v, want file not found", err)
	}
	if _, err := source.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("failed OpenFile created a file: Stat error = %v", err)
	}
}

type regexpStatErrorFs struct {
	Fs
	err error
}

func (f regexpStatErrorFs) Stat(string) (os.FileInfo, error) {
	return nil, f.err
}

func TestRegexpFsOpenFileStatError(t *testing.T) {
	for _, wantErr := range []error{os.ErrPermission, errors.New("stat failed")} {
		t.Run(wantErr.Error(), func(t *testing.T) {
			source := NewMemMapFs()
			filter := NewRegexpFs(
				regexpStatErrorFs{Fs: source, err: wantErr},
				regexp.MustCompile(`\.yaml$`),
			)
			file, err := filter.OpenFile("config.yaml", os.O_CREATE|os.O_WRONLY, 0o600)
			if file != nil {
				file.Close()
			}
			if !errors.Is(err, wantErr) {
				t.Fatalf("OpenFile error = %v, want %v", err, wantErr)
			}
			if _, err := source.Stat("config.yaml"); !os.IsNotExist(err) {
				t.Fatalf("failed OpenFile created a file: Stat error = %v", err)
			}
		})
	}
}
