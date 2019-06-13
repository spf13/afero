// Copyright ©2018 Steve Francia <spf@spf13.com>
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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLstatIfPossible(t *testing.T) {
	wd, _ := os.Getwd()
	defer func() {
		os.Chdir(wd)
	}()

	lt := setupLinkTestData(t, "afero-lstat")

	defer func() {
		lt.osFs.RemoveAll(lt.workDir)
	}()

	checkLstat := func(l Lstater, name string, shouldLstat bool) os.FileInfo {
		statFile, isLstat, err := l.LstatIfPossible(name)
		if err != nil {
			t.Fatalf("Lstat check failed: %s", err)
		}
		if isLstat != shouldLstat {
			t.Fatalf("Lstat status was %t for %s", isLstat, name)
		}
		return statFile
	}

	for idx, td := range []struct {
		fs           Fs
		pathf, paths string
	}{
		{lt.osFs, lt.pathFile, lt.pathSymlink},
		{lt.overlayFs1, lt.pathFile, lt.pathSymlink},
		{lt.overlayFs2, lt.pathFile, lt.pathSymlink},
		{lt.basePathFs, "afero.txt", "symafero.txt"},
		{lt.overlayFsMemOnly, lt.pathFileMem, ""},
		{lt.basePathFsMem, "aferom.txt", ""},
		{lt.roFs, lt.pathFile, lt.pathSymlink},
		{lt.roFsMem, lt.pathFileMem, ""},
	} {
		t.Run(fmt.Sprint(idx), func(t *testing.T) {
			l := td.fs.(Lstater)
			shouldLstat := td.paths != ""
			statRegular := checkLstat(l, td.pathf, shouldLstat)
			statSymlink := checkLstat(l, td.paths, shouldLstat)
			if statRegular == nil || statSymlink == nil {
				t.Fatal("got nil FileInfo")
			}

			symSym := statSymlink.Mode()&os.ModeSymlink == os.ModeSymlink
			if symSym == (td.paths == "") {
				t.Fatal("expected the FileInfo to describe the symlink")
			}

			_, _, err := l.LstatIfPossible("this-should-not-exist.txt")
			if err == nil || !os.IsNotExist(err) {
				t.Fatalf("expected file to not exist, got %s", err)
			}
		})
	}
}

func TestSymlinkIfPossible(t *testing.T) {
	wd, _ := os.Getwd()
	defer func() {
		os.Chdir(wd)
	}()

	lt := setupLinkTestData(t, "afero-symlink")

	defer func() {
		lt.osFs.RemoveAll(lt.workDir)
	}()
	for _, td := range []struct {
		name                 string
		createFs, checkFs    Fs
		pathf, paths         string
		expectLinkPathDiffer bool
	}{
		{
			name:     "1",
			createFs: lt.osFs,
			checkFs:  lt.osFs,
			pathf:    lt.pathFile,
			paths:    lt.pathFile + "-sym",
		},
		{
			name:     "2",
			createFs: lt.osFs,
			pathf:    lt.pathFile,
			paths:    lt.pathFile + "-sym2",
		},
		{
			name:     "3",
			createFs: lt.memFs,
			checkFs:  lt.memFs,
		},
		{
			name:     "4",
			createFs: lt.overlayFs1,
			pathf:    lt.pathFile,
			paths:    lt.pathFile + "-sym4",
		},
		{
			name:     "5",
			createFs: lt.overlayFs2,
			checkFs:  lt.osFs,
			pathf:    lt.pathFile,
			paths:    lt.pathFile + "-sym5",
		},
		{
			name:     "6",
			createFs: lt.overlayFsMemOnly,
			checkFs:  lt.osFs,
			pathf:    lt.pathFile,
		},
		{
			name:     "7",
			createFs: lt.basePathFs,
			checkFs:  lt.basePathFs,
			pathf:    "afero.txt",
			paths:    "afero.txt-sym7",
		},
		{
			name:                 "8",
			createFs:             lt.basePathFs,
			checkFs:              lt.osFs,
			pathf:                "afero.txt",
			paths:                "afero.txt-sym8",
			expectLinkPathDiffer: true,
		},
		{
			name:     "9",
			createFs: lt.basePathFsMem,
			checkFs:  lt.osFs,
			pathf:    "afero.txt",
		},
		{
			name:     "10",
			createFs: lt.roFs,
			checkFs:  lt.osFs,
		},
		{
			name:     "11",
			createFs: lt.roFsMem,
			checkFs:  lt.osFs,
		},
	} {
		t.Run(td.name, func(t *testing.T) {
			sl, ok := td.createFs.(Symlinker)
			if !ok {
				if td.pathf != "" {
					t.Errorf("fs does not allow symlinking")
				}
				return
			} else if td.pathf == "" {
				t.Errorf("fs does allow symlinking, but no file given")
				return
			}
			b, err := sl.SymlinkIfPossible(td.pathf, td.paths)
			if !b && td.paths != "" {
				t.Errorf("given symlink, but unable to symlink")
			}
			if err != nil {
				t.Error(err)
			}
			if !b {
				return
			}
			if !t.Failed() {
				if td.checkFs == nil {
					//look at raw fs (pkg os)
					fi, err := os.Lstat(td.paths)
					if err != nil {
						t.Error(err)
					}
					if fi.Mode()&os.ModeSymlink == 0 {
						t.Errorf("not a symlink")
					}
				} else {
					ls := td.checkFs.(Lstater)
					fi, b, err := ls.LstatIfPossible(td.paths)
					if !b {
						t.Errorf("lstat failed")
					} else if err != nil {
						t.Error(err)
					} else if fi.Mode()&os.ModeSymlink == 0 {
						t.Errorf("not a symlink: %#v", fi)
					}
					path, b, err := EvalSymlinks(td.checkFs, td.paths)
					if !b {
						t.Errorf("EvalSymlinks failed")
					} else if err != nil {
						t.Error(err)
					} else if td.expectLinkPathDiffer != (path != td.pathf) {
						t.Errorf("path anomaly - got %s, had %s, expectLinkPathDiffer=%t", path, td.pathf, td.expectLinkPathDiffer)
					}
				}
			}
			if t.Failed() && runtime.GOOS == "windows" && (td.name == "1" || td.name == "5") {
				t.Log("NOTE - tests 1, 5 require additional privileges on Windows.")
				t.Log("These privileges require Win10 Creators' Update + dev mode,")
				t.Log("and/or running tests from the administrator command prompt.")
			}
		})
	}
}

func TestReadlinkIfPossible(t *testing.T) {
	wd, _ := os.Getwd()
	defer func() {
		os.Chdir(wd)
	}()

	lt := setupLinkTestData(t, "afero-readlink")

	defer func() {
		lt.osFs.RemoveAll(lt.workDir)
	}()
	for _, td := range []struct {
		name                 string
		fs                   Fs
		pathf, paths         string
		expectLinkPathDiffer bool
	}{
		{
			name:  "os",
			fs:    lt.osFs,
			pathf: "afero.txt",
			paths: lt.pathSymlink,
		},
		{
			name:  "overlay1",
			fs:    lt.overlayFs1,
			pathf: "afero.txt",
			paths: lt.pathSymlink,
		},
		{
			name:  "overlay2",
			fs:    lt.overlayFs2,
			pathf: "afero.txt",
			paths: lt.pathSymlink,
		},
		{
			name:  "base",
			fs:    lt.basePathFs,
			pathf: "afero.txt",
			paths: "symafero.txt",
		},
		{
			name: "overlayMem",
			fs:   lt.overlayFsMemOnly,
		},
		{
			name: "baseMem",
			fs:   lt.basePathFsMem,
		},
		{
			name:  "ro",
			fs:    lt.roFs,
			pathf: "afero.txt",
			paths: lt.pathSymlink,
		},
		{
			name: "roMem",
			fs:   lt.roFsMem,
		},
	} {
		t.Run(td.name, func(t *testing.T) {
			rl, ok := td.fs.(Readlinker)
			if !ok && td.paths == "" {
				t.Logf("Readlinker not available with %s - success.", td.fs.Name())
				return
			}
			if !ok {
				t.Error("not readlinker")
			}
			link, poss, err := rl.ReadlinkIfPossible(td.paths)
			if !poss && td.pathf != "" {
				t.Error("not possible")
			}
			if !poss {
				t.Logf("Readlink is not possible on %s - success.", td.fs.Name())
				return
			}
			if err != nil {
				t.Error(err)
			}
			if td.paths == "" {
				t.Errorf("Readlink should not be possible on %s", td.fs.Name())
			}
			if td.expectLinkPathDiffer {
				if link == td.pathf {
					t.Errorf("expect different paths but both are %s", link)
				}
			} else {
				if link != td.pathf {
					t.Errorf("got %s, want %s", link, td.pathf)
				}
			}
		})
	}
}

type linkTestData struct {
	workDir, memWorkDir, pathFileMem    string
	pathFile, pathSymlink               string
	osFs, memFs, overlayFs1, overlayFs2 Fs
	overlayFsMemOnly, basePathFs        Fs
	basePathFsMem, roFs, roFsMem        Fs
}

func setupLinkTestData(t *testing.T, dirpfx string) linkTestData {
	t.Helper()
	var err error
	var lt linkTestData

	lt.osFs = &OsFs{}

	lt.workDir, err = TempDir(lt.osFs, "", dirpfx)
	if err != nil {
		t.Fatal(err)
	}

	lt.memWorkDir = "/" + dirpfx
	lt.memFs = NewMemMapFs()
	lt.overlayFs1 = &CopyOnWriteFs{base: lt.osFs, layer: lt.memFs}
	lt.overlayFs2 = &CopyOnWriteFs{base: lt.memFs, layer: lt.osFs}
	lt.overlayFsMemOnly = &CopyOnWriteFs{base: lt.memFs, layer: NewMemMapFs()}
	lt.basePathFs = &BasePathFs{source: lt.osFs, path: lt.workDir}
	lt.basePathFsMem = &BasePathFs{source: lt.memFs, path: lt.memWorkDir}
	lt.roFs = &ReadOnlyFs{source: lt.osFs}
	lt.roFsMem = &ReadOnlyFs{source: lt.memFs}

	lt.pathFileMem = filepath.Join(lt.memWorkDir, "aferom.txt")

	WriteFile(lt.osFs, filepath.Join(lt.workDir, "afero.txt"), []byte("Hi, Afero!"), 0777)
	WriteFile(lt.memFs, filepath.Join(lt.pathFileMem), []byte("Hi, Afero!"), 0777)

	os.Chdir(lt.workDir)
	if err = os.Symlink("afero.txt", "symafero.txt"); err != nil {
		t.Fatal(err)
	}

	lt.pathFile = filepath.Join(lt.workDir, "afero.txt")
	lt.pathSymlink = filepath.Join(lt.workDir, "symafero.txt")

	return lt
}

func TestEvalSymlinks(t *testing.T) {
	wd, _ := os.Getwd()
	defer func() {
		os.Chdir(wd)
	}()

	lt := setupLinkTestData(t, "afero-evalsym")

	defer func() {
		lt.osFs.RemoveAll(lt.workDir)
	}()

	//does this get all corner cases?
	testdata := []struct {
		name      string
		fs        Fs
		fspath    string
		want      string
		b         bool
		expectErr bool
	}{
		{name: "os", fs: lt.osFs, fspath: lt.pathFile, want: lt.pathFile, b: true},
		{name: "oss", fs: lt.osFs, fspath: lt.pathSymlink, want: lt.pathFile, b: true},
		{name: "o1", fs: lt.overlayFs1, fspath: lt.pathFile, want: lt.pathFile, b: true},
		{name: "o1s", fs: lt.overlayFs1, fspath: lt.pathSymlink, want: lt.pathFile, b: true},
		{name: "o2", fs: lt.overlayFs2, fspath: lt.pathFile, want: lt.pathFile, b: true},
		{name: "o2s", fs: lt.overlayFs2, fspath: lt.pathSymlink, want: lt.pathFile, b: true},
		{name: "b", fs: lt.basePathFs, fspath: "afero.txt", want: "afero.txt", b: true},
		{name: "bs", fs: lt.basePathFs, fspath: "symafero.txt", want: "afero.txt", b: true},
		{name: "om", fs: lt.overlayFsMemOnly, fspath: lt.pathFileMem, b: false},
		{name: "bm", fs: lt.basePathFsMem, fspath: lt.pathFileMem, b: false},
		{name: "ro", fs: lt.roFs, fspath: lt.pathFile, want: lt.pathFile, b: true},
		{name: "rom", fs: lt.roFsMem, fspath: lt.pathFile, b: false},
	}
	for _, td := range testdata {
		t.Run(td.name, func(t *testing.T) {
			got, b, err := EvalSymlinks(td.fs, td.fspath)
			if b != td.b {
				t.Errorf("b mismatch - got %t want %t", b, td.b)
			}
			if !b {
				return
			}
			if td.expectErr {
				if err == nil {
					t.Errorf("expected error, got none")
				}
				return
			}
			if err != nil {
				t.Error(err)
			}
			if got != td.want {
				t.Errorf("mismatch '%s' vs '%s'", got, td.want)
			}
		})
	}
}

func TestResolveRelative(t *testing.T) {
	testdata := []struct {
		name             string
		path, link, want string
		expectErr        bool
	}{
		{
			name:      "1",
			path:      "/test/path",
			link:      "../a//b",
			want:      "/test/a/b",
			expectErr: false,
		},
		{
			name:      "2",
			path:      "/test/path",
			link:      "../a/./b",
			want:      "/test/a/b",
			expectErr: false,
		},
		{
			name:      "3",
			path:      "/test/path",
			link:      "../a/../b",
			want:      "/test/b",
			expectErr: false,
		},
		{
			name:      "4",
			path:      "/test/path",
			link:      "../../../a",
			want:      "",
			expectErr: true,
		},
		{
			name:      "5",
			path:      "/test/path",
			link:      ".//a",
			want:      "/test/path/a",
			expectErr: false,
		},
		{
			name:      "6",
			path:      "/test/path",
			link:      "a//",
			want:      "/test/path/a/",
			expectErr: false,
		},
		{
			name:      "7",
			path:      "",
			link:      "a",
			want:      "a",
			expectErr: false,
		},
	}
	for _, td := range testdata {
		t.Run(td.name, func(t *testing.T) {
			if runtime.GOOS == "windows" {
				td.path = strings.Replace(td.path, "/", "\\", -1)
				td.link = strings.Replace(td.link, "/", "\\", -1)
				td.want = strings.Replace(td.want, "/", "\\", -1)
			}
			got, err := resolveRelative(td.path, td.link)
			if td.expectErr != (err != nil) {
				t.Errorf("expect error: %t; got error: %s", td.expectErr, err)
			}
			if got != td.want {
				t.Errorf("got %s, want %s", got, td.want)
			}
		})
	}
}
