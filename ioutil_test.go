// ©2015 The Go Authors
// Copyright ©2015 Steve Francia <spf@spf13.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package afero

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func checkSizePath(t *testing.T, path string, size int64) {
	dir, err := testFS.Stat(path)
	if err != nil {
		t.Fatalf("Stat %q (looking for size %d): %s", path, size, err)
	}
	if dir.Size() != size {
		t.Errorf("Stat %q: size %d want %d", path, dir.Size(), size)
	}
}

func TestReadFile(t *testing.T) {
	testFS = &MemMapFs{}
	fsutil := &Afero{Fs: testFS}

	testFS.Create("this_exists.go")
	filename := "rumpelstilzchen"
	_, err := fsutil.ReadFile(filename)
	if err == nil {
		t.Fatalf("ReadFile %s: error expected, none found", filename)
	}

	filename = "this_exists.go"
	contents, err := fsutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", filename, err)
	}

	checkSizePath(t, filename, int64(len(contents)))
}

func TestWriteFile(t *testing.T) {
	testFS = &MemMapFs{}
	fsutil := &Afero{Fs: testFS}
	f, err := fsutil.TempFile("", "ioutil-test")
	if err != nil {
		t.Fatal(err)
	}
	filename := f.Name()
	data := "Programming today is a race between software engineers striving to " +
		"build bigger and better idiot-proof programs, and the Universe trying " +
		"to produce bigger and better idiots. So far, the Universe is winning."

	if err := fsutil.WriteFile(filename, []byte(data), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", filename, err)
	}

	contents, err := fsutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", filename, err)
	}

	if string(contents) != data {
		t.Fatalf("contents = %q\nexpected = %q", string(contents), data)
	}

	// cleanup
	f.Close()
	testFS.Remove(filename) // ignore error
}

func TestReadDir(t *testing.T) {
	testFS = &MemMapFs{}
	testFS.Mkdir("/i-am-a-dir", 0o777)
	testFS.Create("/this_exists.go")
	dirname := "rumpelstilzchen"
	_, err := ReadDir(testFS, dirname)
	if err == nil {
		t.Fatalf("ReadDir %s: error expected, none found", dirname)
	}

	dirname = ".."
	list, err := ReadDir(testFS, dirname)
	if err != nil {
		t.Fatalf("ReadDir %s: %v", dirname, err)
	}

	foundFile := false
	foundSubDir := false
	for _, dir := range list {
		switch {
		case !dir.IsDir() && dir.Name() == "this_exists.go":
			foundFile = true
		case dir.IsDir() && dir.Name() == "i-am-a-dir":
			foundSubDir = true
		}
	}
	if !foundFile {
		t.Fatalf("ReadDir %s: this_exists.go file not found", dirname)
	}
	if !foundSubDir {
		t.Fatalf("ReadDir %s: i-am-a-dir directory not found", dirname)
	}
}

func TestTempFile(t *testing.T) {
	type args struct {
		dir     string
		pattern string
	}
	tests := map[string]struct {
		args args
		want func(*testing.T, string)
	}{
		"foo": { // simple file name
			args: args{
				dir:     "",
				pattern: "foo",
			},
			want: func(t *testing.T, base string) {
				if !strings.HasPrefix(base, "foo") || len(base) <= len("foo") {
					t.Errorf("TempFile() file = %s, invalid file name", base)
				}
			},
		},
		"foo.bar": { // file name w/ ext
			args: args{
				dir:     "",
				pattern: "foo.bar",
			},
			want: func(t *testing.T, base string) {
				if !strings.HasPrefix(base, "foo.bar") || len(base) <= len("foo.bar") {
					t.Errorf("TempFile() file = %v, invalid file name", base)
				}
			},
		},
		"foo-*.bar": { // file name with wild card
			args: args{
				dir:     "",
				pattern: "foo-*.bar",
			},
			want: func(t *testing.T, base string) {
				//nolint: staticcheck
				if !(strings.HasPrefix(base, "foo-") || strings.HasPrefix(base, "bar")) ||
					len(base) <= len("foo-*.bar") {
					t.Errorf("TempFile() file = %v, invalid file name", base)
				}
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			file, err := TempFile(NewMemMapFs(), tt.args.dir, tt.args.pattern)
			if err != nil {
				t.Errorf("TempFile() error = %v, none expected", err)
				return
			}
			if file == nil {
				t.Errorf("TempFile() file = %v, should not be nil", file)
				return
			}
			tt.want(t, filepath.Base(file.Name()))
		})
	}
}

func TestTempDir(t *testing.T) {

	type testData struct {
		dir            string
		prefix         string
		expectedPrefix string
		shouldError    bool
	}

	tests := map[string]testData{
		"TempDirWithStar": {
			dir:            "",
			prefix:         "withStar*",
			expectedPrefix: "/tmp/withStar",
		},
		"TempDirWithTwoStars": {
			dir:            "",
			prefix:         "withStar**",
			expectedPrefix: "/tmp/withStar*",
		},
		"TempDirWithoutStar": {
			dir:            "",
			prefix:         "withoutStar",
			expectedPrefix: "/tmp/withoutStar",
		},
		"UserDir": {
			dir:            "dir1",
			prefix:         "",
			expectedPrefix: "dir1/",
		},
		"InvalidPrefix": {
			dir:            "",
			prefix:         "hello/world",
			expectedPrefix: "hello",
			shouldError:    true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			afs := NewMemMapFs()
			result, err := TempDir(afs, test.dir, test.prefix)

			if test.shouldError {
				if err == nil {
					t.Error("err should not be nil")
					return
				}
				if result != "" {
					t.Errorf("result was %s and should be the empty string", result)
					return
				}
			} else {
				match, _ := regexp.MatchString(test.expectedPrefix+".*", result)
				if !match {
					t.Errorf("directory should have prefix %s should exist, but doesn't", test.expectedPrefix)
					return
				}
				exists, err := DirExists(afs, result)
				if !exists {
					t.Errorf("directory with prefix %s should exist, but doesn't", test.expectedPrefix)
					return
				}
				if err != nil {
					t.Errorf("err should be nil, but was %s", err.Error())
					return
				}
			}

			afs.RemoveAll(result)
		})
	}
}
