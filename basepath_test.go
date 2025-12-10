package afero

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBasePath(t *testing.T) {
	baseFs := &MemMapFs{}
	baseFs.MkdirAll("/base/path/tmp", 0o777)
	bp := NewBasePathFs(baseFs, "/base/path")

	if _, err := bp.Create("/tmp/foo"); err != nil {
		t.Errorf("Failed to set real path")
	}

	if fh, err := bp.Create("../tmp/bar"); err == nil {
		t.Errorf("succeeded in creating %s ...", fh.Name())
	}
}

func TestBasePathRoot(t *testing.T) {
	baseFs := &MemMapFs{}
	baseFs.MkdirAll("/base/path/foo/baz", 0o777)
	baseFs.MkdirAll("/base/path/boo/", 0o777)
	bp := NewBasePathFs(baseFs, "/base/path")

	rd, err := ReadDir(bp, string(os.PathSeparator))

	if len(rd) != 2 {
		t.Errorf("base path doesn't respect root")
	}

	if err != nil {
		t.Error(err)
	}
}

func TestRealPath(t *testing.T) {
	tests := []struct {
		name      string
		base      string
		input     string
		want      string
		expectErr bool
		os        string
	}{
		// 1. Happy Paths
		{
			name:  "Simple subpath",
			base:  "/var/data",
			input: "file.txt",
			want:  "/var/data/file.txt",
		},
		{
			name:  "Deeply nested subpath",
			base:  "/var/data",
			input: "subdir/images/logo.png",
			want:  "/var/data/subdir/images/logo.png",
		},

		// 2. Cleaning Behavior
		{
			name:  "Cleans double slashes",
			base:  "/var/data",
			input: "subdir//file.txt",
			want:  "/var/data/subdir/file.txt",
		},
		{
			name:  "Cleans current directory dots",
			base:  "/var/data",
			input: "./subdir/./file.txt",
			want:  "/var/data/subdir/file.txt",
		},
		{
			name:  "Cleans subpath starts with /",
			base:  "/var/data",
			input: "/file.txt",
			want:  "/var/data/file.txt",
		},
		{
			name:  "Resolves internal dot-dot (safe)",
			base:  "/var/data",
			input: "subdir/../file.txt",
			want:  "/var/data/file.txt",
		},
		{
			name:  "Resolves base path dot-dot",
			base:  "/var/data/../data",
			input: "file.txt",
			want:  "/var/data/file.txt",
		},

		// 3. Base Path is "."
		{
			name:  "Base is dot, simple file",
			base:  ".",
			input: "file.txt",
			want:  "file.txt",
		},
		{
			name:  "Base is dot, input has dot prefix",
			base:  ".",
			input: "./file.txt",
			want:  "file.txt",
		},
		{
			name:  "Base is dot, safe traversal",
			base:  ".",
			input: "foo/../bar",
			want:  "bar",
		},

		// 4. paths starting with ..
		{
			name:  "Valid file starting with .. (..X)",
			base:  "/var/data",
			input: "..foo",
			want:  "/var/data/..foo",
		},
		{
			name:  "Valid file named ...",
			base:  "/var/data",
			input: "...",
			want:  "/var/data/...",
		},
		{
			name:  "Hidden file",
			base:  "/var/data",
			input: ".config",
			want:  "/var/data/.config",
		},
		{
			name:  "Base is dot, input is ..foo",
			base:  ".",
			input: "..foo",
			want:  "..foo",
		},

		// 5. Failure Cases
		{
			name:      "Traversal out (parent)",
			base:      "/var/data",
			input:     "../etc/passwd",
			expectErr: true,
		},
		{
			name:      "Traversal out (root)",
			base:      "/var/data",
			input:     "../../../../etc/passwd",
			expectErr: true,
		},
		{
			name:      "Base is dot, traversal out",
			base:      ".",
			input:     "../file.txt",
			expectErr: true,
		},
		{
			name:      "Partial suffix match (e.g. /var/dataset vs /var/data)",
			base:      "/var/data",
			input:     "../dataset/file.txt",
			expectErr: true,
		},
		{
			name:      "Windows: Absolute path",
			base:      `C:\base`,
			input:     `C:\Windows\System32`,
			expectErr: true,
			os:        "windows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.os != "" && tt.os != runtime.GOOS {
				t.Skipf("Skipping test for OS %q", tt.os)
			}

			baseFs := &MemMapFs{}
			bpInterface := NewBasePathFs(baseFs, tt.base)
			bp := bpInterface.(*BasePathFs)

			got, err := bp.RealPath(tt.input)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error for input %q, got nil. Result was: %q", tt.input, got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tt.input, err)
			}

			if runtime.GOOS == "windows" {
				tt.want = filepath.FromSlash(tt.want)
			}

			if got != tt.want {
				t.Errorf("RealPath() mismatch.\nBase: %q\nInput: %q\nGot:  %q\nWant: %q",
					tt.base, tt.input, got, tt.want)
			}
		})
	}
}

func TestNestedBasePaths(t *testing.T) {
	type dirSpec struct {
		Dir1, Dir2, Dir3 string
	}
	dirSpecs := []dirSpec{
		{Dir1: "/", Dir2: "/", Dir3: "/"},
		{Dir1: "/", Dir2: "/path2", Dir3: "/"},
		{Dir1: "/path1/dir", Dir2: "/path2/dir/", Dir3: "/path3/dir"},
		{Dir1: "C:/path1", Dir2: "path2/dir", Dir3: "/path3/dir/"},
	}

	for _, ds := range dirSpecs {
		memFs := NewMemMapFs()
		level1Fs := NewBasePathFs(memFs, ds.Dir1)
		level2Fs := NewBasePathFs(level1Fs, ds.Dir2)
		level3Fs := NewBasePathFs(level2Fs, ds.Dir3)

		type spec struct {
			BaseFs   Fs
			FileName string
		}
		specs := []spec{
			{BaseFs: level3Fs, FileName: "f.txt"},
			{BaseFs: level2Fs, FileName: "f.txt"},
			{BaseFs: level1Fs, FileName: "f.txt"},
		}

		for _, s := range specs {
			if err := s.BaseFs.MkdirAll(s.FileName, 0o755); err != nil {
				t.Errorf("Got error %s", err.Error())
			}
			if _, err := s.BaseFs.Stat(s.FileName); err != nil {
				t.Errorf("Got error %s", err.Error())
			}

			switch s.BaseFs {
			case level3Fs:
				pathToExist := filepath.Join(ds.Dir3, s.FileName)
				if _, err := level2Fs.Stat(pathToExist); err != nil {
					t.Errorf("Got error %s (path %s)", err.Error(), pathToExist)
				}

			case level2Fs:
				pathToExist := filepath.Join(ds.Dir2, ds.Dir3, s.FileName)
				if _, err := level1Fs.Stat(pathToExist); err != nil {
					t.Errorf("Got error %s (path %s)", err.Error(), pathToExist)
				}
			}
		}
	}
}

func TestBasePathOpenFile(t *testing.T) {
	baseFs := &MemMapFs{}
	baseFs.MkdirAll("/base/path/tmp", 0o777)
	bp := NewBasePathFs(baseFs, "/base/path")
	f, err := bp.OpenFile("/tmp/file.txt", os.O_CREATE, 0o600)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	if filepath.Dir(f.Name()) != filepath.Clean("/tmp") {
		t.Fatalf("realpath leaked: %s", f.Name())
	}
}

func TestBasePathCreate(t *testing.T) {
	baseFs := &MemMapFs{}
	baseFs.MkdirAll("/base/path/tmp", 0o777)
	bp := NewBasePathFs(baseFs, "/base/path")
	f, err := bp.Create("/tmp/file.txt")
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if filepath.Dir(f.Name()) != filepath.Clean("/tmp") {
		t.Fatalf("realpath leaked: %s", f.Name())
	}
}

func TestBasePathTempFile(t *testing.T) {
	baseFs := &MemMapFs{}
	baseFs.MkdirAll("/base/path/tmp", 0o777)
	bp := NewBasePathFs(baseFs, "/base/path")

	tDir, err := TempDir(bp, "/tmp", "")
	if err != nil {
		t.Fatalf("Failed to TempDir: %v", err)
	}
	if filepath.Dir(tDir) != filepath.Clean("/tmp") {
		t.Fatalf("Tempdir realpath leaked: %s", tDir)
	}
	tempFile, err := TempFile(bp, tDir, "")
	if err != nil {
		t.Fatalf("Failed to TempFile: %v", err)
	}
	defer tempFile.Close()
	if expected, actual := tDir, filepath.Dir(tempFile.Name()); expected != actual {
		t.Fatalf("TempFile realpath leaked: expected %s, got %s", expected, actual)
	}
}
