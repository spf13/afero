// Copyright © 2014 Steve Francia <spf@spf13.com>.
// Copyright 2009 The Go Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package afero

import (
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestWalk(t *testing.T) {
	defer removeAllTestFiles(t)
	var testDir string
	for i, fs := range Fss {
		if i == 0 {
			testDir = setupTestDirRoot(t, fs)
		} else {
			setupTestDirReusePath(t, fs, testDir)
		}
	}

	outputs := make([]string, len(Fss))
	for i, fs := range Fss {
		walkFn := func(path string, info os.FileInfo, err error) error {
			if err != nil {
				t.Error("walkFn err:", err)
			}
			var size int64
			if !info.IsDir() {
				size = info.Size()
			}
			outputs[i] += fmt.Sprintln(path, info.Name(), size, info.IsDir(), err)
			return nil
		}
		err := Walk(fs, testDir, walkFn)
		if err != nil {
			t.Error(err)
		}
	}
	fail := false
	for i, o := range outputs {
		if i == 0 {
			continue
		}
		if o != outputs[i-1] {
			fail = true
			break
		}
	}
	if fail {
		t.Log("Walk outputs not equal!")
		for i, o := range outputs {
			t.Log(Fss[i].Name() + "\n" + o)
		}
		t.Fail()
	}
}

func TestWalkDir(t *testing.T) {
	defer removeAllTestFiles(t)

	var testDir string

	for i, fs := range Fss {
		if i == 0 {
			testDir = setupTestDirRoot(t, fs)

			continue
		}

		setupTestDirReusePath(t, fs, testDir)
	}

	outputs := make([]string, len(Fss))

	for i, fs := range Fss {
		walkDirFn := func(path string, d iofs.DirEntry, err error) error {
			if err != nil {
				t.Error("walkDirFn err:", err)
			}

			var size int64

			if !d.IsDir() {
				info, infoErr := d.Info()
				if infoErr != nil {
					t.Error("d.Info() err:", infoErr)
				}

				size = info.Size()
			}

			outputs[i] += fmt.Sprintln(path, d.Name(), size, d.IsDir(), err)

			return nil
		}

		err := WalkDir(fs, testDir, walkDirFn)
		if err != nil {
			t.Error(err)
		}
	}

	fail := false

	for i, o := range outputs {
		if i == 0 {
			continue
		}

		if o != outputs[i-1] {
			fail = true

			break
		}
	}
	if fail {
		t.Log("WalkDir outputs not equal!")

		for i, o := range outputs {
			t.Log(Fss[i].Name() + "\n" + o)
		}

		t.Fail()
	}
}

func TestWalkDirSkipDir(t *testing.T) {
	defer removeAllTestFiles(t)

	for _, fs := range Fss {
		root := testDir(fs)
		fs.MkdirAll(filepath.Join(root, "more", "subdirectories"), 0o700)
		WriteFile(fs, filepath.Join(root, "more", "subdirectories", "file.txt"), []byte("hello"), 0o644)
		WriteFile(fs, filepath.Join(root, "other.txt"), []byte("world"), 0o644)

		var visited []string

		walkDirFn := func(path string, d iofs.DirEntry, err error) error {
			if err != nil {
				t.Error("walkDirFn err:", err)
			}

			rel, _ := filepath.Rel(root, path)
			visited = append(visited, rel)

			if d.IsDir() && d.Name() == "more" {
				return filepath.SkipDir
			}

			return nil
		}
		err := WalkDir(fs, root, walkDirFn)
		if err != nil {
			t.Error(fs.Name(), err)
		}

		foundMore := false

		for _, v := range visited {
			if v == "more" {
				foundMore = true
			}

			if v == filepath.Join("more", "subdirectories") {
				t.Errorf("%s: should not have visited more/subdirectories", fs.Name())
			}
		}

		if !foundMore {
			t.Errorf("%s: should have visited 'more'", fs.Name())
		}
	}
}

func TestWalkDirSkipAll(t *testing.T) {
	defer removeAllTestFiles(t)

	for _, fs := range Fss {
		root := testDir(fs)

		WriteFile(fs, filepath.Join(root, "a.txt"), []byte("a"), 0o644)
		WriteFile(fs, filepath.Join(root, "b.txt"), []byte("b"), 0o644)
		WriteFile(fs, filepath.Join(root, "c.txt"), []byte("c"), 0o644)

		count := 0
		walkDirFn := func(path string, d iofs.DirEntry, err error) error {
			count++
			if count >= 2 {
				return filepath.SkipAll
			}

			return nil
		}

		err := WalkDir(fs, root, walkDirFn)
		if err != nil {
			t.Error(fs.Name(), err)
		}

		if count > 2 {
			t.Errorf("%s: expected at most 2 entries visited, got %d", fs.Name(), count)
		}
	}
}

func TestWalkDirError(t *testing.T) {
	defer removeAllTestFiles(t)

	for _, fs := range Fss {
		var callbackErr error

		walkDirFn := func(path string, d iofs.DirEntry, err error) error {
			callbackErr = err
			return err
		}

		err := WalkDir(fs, "/nonexistent-path-for-walkdir-test", walkDirFn)
		if err == nil {
			t.Errorf("%s: expected error for nonexistent root", fs.Name())
		}

		if callbackErr == nil {
			t.Errorf("%s: expected callback to receive error", fs.Name())
		}
	}
}
