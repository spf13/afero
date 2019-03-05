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
	"os"
	"path/filepath"
	"testing"
)

func TestEvalSymlinksIfPossible(t *testing.T) {
	wd, _ := os.Getwd()
	defer func() {
		os.Chdir(wd)
	}()

	osFs := &OsFs{}

	workDir, err := TempDir(osFs, "", "afero-eval-symlinks")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		osFs.RemoveAll(workDir)
	}()

	memWorkDir := "/symlinks"

	memFs := NewMemMapFs()
	overlayFs1 := &CopyOnWriteFs{base: osFs, layer: memFs}
	overlayFs2 := &CopyOnWriteFs{base: memFs, layer: osFs}
	overlayFsMemOnly := &CopyOnWriteFs{base: memFs, layer: NewMemMapFs()}
	basePathFs := &BasePathFs{source: osFs, path: workDir}
	basePathFsMem := &BasePathFs{source: memFs, path: memWorkDir}
	roFs := &ReadOnlyFs{source: osFs}
	roFsMem := &ReadOnlyFs{source: memFs}

	pathFileMem := filepath.Join(memWorkDir, "aferom.txt")

	WriteFile(osFs, filepath.Join(workDir, "afero.txt"), []byte("Hi, Afero!"), 0777)
	WriteFile(memFs, filepath.Join(pathFileMem), []byte("Hi, Afero!"), 0777)

	os.Chdir(workDir)
	if err := os.Symlink("afero.txt", "symafero.txt"); err != nil {
		t.Fatal(err)
	}

	pathFile := filepath.Join(workDir, "afero.txt")
	pathSymlink := filepath.Join(workDir, "symafero.txt")

	checkEvalSymlinks := func(s Symlinker, path string, shouldEvalSymlink bool) string {
		evalPath, isSymlink, err := s.EvalSymlinksIfPossible(path)
		if err != nil {
			t.Fatalf("EvalSymlinks check failed: %s", err)
		}
		if isSymlink != shouldEvalSymlink {
			t.Fatalf("EvalSymlinks status was %t for %s", isSymlink, path)
		}
		return evalPath
	}

	testEvalSymlinks := func(s Symlinker, pathFile, pathSymlink string) {
		shoouldSymlink := pathSymlink != ""
		evalPathRegular := checkEvalSymlinks(s, pathFile, shoouldSymlink)
		evalPathSymlink := checkEvalSymlinks(s, pathSymlink, shoouldSymlink)
		if shoouldSymlink && (evalPathRegular == "" || evalPathSymlink == "") {
			t.Fatal("got empty path")
		}

		_, _, err := s.EvalSymlinksIfPossible("this-should-not-exist.txt")
		if err == nil || !os.IsNotExist(err) {
			t.Fatalf("expected file to not exist, got %v", err)
		}
	}

	testEvalSymlinks(osFs, pathFile, pathSymlink)
	testEvalSymlinks(overlayFs1, pathFile, pathSymlink)
	testEvalSymlinks(overlayFs2, pathFile, pathSymlink)
	testEvalSymlinks(basePathFs, "afero.txt", "symafero.txt")
	testEvalSymlinks(overlayFsMemOnly, pathFileMem, "")
	testEvalSymlinks(basePathFsMem, "aferom.txt", "")
	testEvalSymlinks(roFs, pathFile, pathSymlink)
	testEvalSymlinks(roFsMem, pathFileMem, "")
}

func TestSymlinkIfPossible(t *testing.T) {
	wd, _ := os.Getwd()
	defer func() {
		os.Chdir(wd)
	}()

	osFs := &OsFs{}

	workDir, err := TempDir(osFs, "", "afero-symlink")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		osFs.RemoveAll(workDir)
	}()

	memWorkDir := "/symlinks"

	memFs := NewMemMapFs()
	overlayFsOsOnly := &CopyOnWriteFs{base: NewMemMapFs(), layer: osFs}
	basePathFs := &BasePathFs{source: osFs, path: workDir}
	basePathFsMem := &BasePathFs{source: memFs, path: memWorkDir}

	pathFileMem := filepath.Join(memWorkDir, "aferom.txt")

	WriteFile(osFs, filepath.Join(workDir, "afero.txt"), []byte("Hi, Afero!"), 0777)
	WriteFile(memFs, filepath.Join(pathFileMem), []byte("Hi, Afero!"), 0777)

	os.Chdir(workDir)

	pathFile := filepath.Join(workDir, "afero.txt")

	testEvalSymlinks := func(s Symlinker, oldPath, newPath string, shouldSymlink bool) {
		isSymlink, err := s.SymlinkIfPossible(oldPath, newPath)
		if err != nil {
			t.Fatalf("Symlink check failed: %s", err)
		}
		if isSymlink != shouldSymlink {
			t.Fatalf("Symlink status was %t for %s %s", isSymlink, oldPath, newPath)
		}
	}

	testEvalSymlinks(osFs, pathFile, "ossym.txt", true)
	testEvalSymlinks(basePathFs, "afero.txt", "basesym.txt", true)
	testEvalSymlinks(overlayFsOsOnly, pathFileMem, "overlayossym.txt", true)
	testEvalSymlinks(basePathFsMem, "aferom.txt", "basepathfsmemsym.txt", false)
}
