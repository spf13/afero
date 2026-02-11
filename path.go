// Copyright ©2015 The Go Authors
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
	iofs "io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/afero/internal/common"
)

// readDirNames reads the directory named by dirname and returns
// a sorted list of directory entries.
// adapted from https://golang.org/src/path/filepath/path.go
func readDirNames(fs Fs, dirname string) ([]string, error) {
	f, err := fs.Open(dirname)
	if err != nil {
		return nil, err
	}
	names, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}

// walk recursively descends path, calling walkFn
// adapted from https://golang.org/src/path/filepath/path.go
func walk(fs Fs, path string, info os.FileInfo, walkFn filepath.WalkFunc) error {
	err := walkFn(path, info, nil)
	if err != nil {
		if info.IsDir() && err == filepath.SkipDir {
			return nil
		}
		return err
	}

	if !info.IsDir() {
		return nil
	}

	names, err := readDirNames(fs, path)
	if err != nil {
		return walkFn(path, info, err)
	}

	for _, name := range names {
		filename := filepath.Join(path, name)
		fileInfo, err := lstatIfPossible(fs, filename)
		if err != nil {
			if err := walkFn(filename, fileInfo, err); err != nil && err != filepath.SkipDir {
				return err
			}
		} else {
			err = walk(fs, filename, fileInfo, walkFn)
			if err != nil {
				if !fileInfo.IsDir() || err != filepath.SkipDir {
					return err
				}
			}
		}
	}
	return nil
}

// if the filesystem supports it, use Lstat, else use fs.Stat
func lstatIfPossible(fs Fs, path string) (os.FileInfo, error) {
	if lfs, ok := fs.(Lstater); ok {
		fi, _, err := lfs.LstatIfPossible(path)
		return fi, err
	}
	return fs.Stat(path)
}

// Walk walks the file tree rooted at root, calling walkFn for each file or
// directory in the tree, including root. All errors that arise visiting files
// and directories are filtered by walkFn. The files are walked in lexical
// order, which makes the output deterministic but means that for very
// large directories Walk can be inefficient.
// Walk does not follow symbolic links.

func (a Afero) Walk(root string, walkFn filepath.WalkFunc) error {
	return Walk(a.Fs, root, walkFn)
}

func Walk(fs Fs, root string, walkFn filepath.WalkFunc) error {
	info, err := lstatIfPossible(fs, root)
	if err != nil {
		return walkFn(root, nil, err)
	}
	return walk(fs, root, info, walkFn)
}

// readDirEntries reads the directory named by dirname and returns
// a sorted list of directory entries.
func readDirEntries(fs Fs, dirname string) ([]iofs.DirEntry, error) {
	f, err := fs.Open(dirname)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []iofs.DirEntry

	if rdf, ok := f.(iofs.ReadDirFile); ok {
		entries, err = rdf.ReadDir(-1)
		if err != nil {
			return nil, err
		}
	} else {
		var infos []os.FileInfo

		infos, err = f.Readdir(-1)
		if err != nil {
			return nil, err
		}

		entries = make([]iofs.DirEntry, len(infos))

		for i, info := range infos {
			entries[i] = common.FileInfoDirEntry{FileInfo: info}
		}
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	return entries, nil
}

// walkDir recursively descends path, calling walkDirFn.
// adapted from https://go.dev/src/path/filepath/path.go
func walkDir(fs Fs, path string, d iofs.DirEntry, walkDirFn iofs.WalkDirFunc) error {
	if err := walkDirFn(path, d, nil); err != nil || !d.IsDir() {
		if err == filepath.SkipDir && d.IsDir() {
			err = nil
		}

		return err
	}

	entries, err := readDirEntries(fs, path)
	if err != nil {
		err = walkDirFn(path, d, err)
		if err != nil {
			if err == filepath.SkipDir && d.IsDir() {
				err = nil
			}

			return err
		}
	}

	for _, entry := range entries {
		name := filepath.Join(path, entry.Name())
		if err := walkDir(fs, name, entry, walkDirFn); err != nil {
			if err == filepath.SkipDir {
				break
			}

			return err
		}
	}
	return nil
}

// WalkDir walks the file tree rooted at root, calling fn for each file or
// directory in the tree, including root. The fn callback receives an fs.DirEntry
// instead of os.FileInfo, which can be more efficient since it does not require
// a stat call for every visited file.
//
// All errors that arise visiting files and directories are filtered by fn:
// see the fs.WalkDirFunc documentation for details.
//
// The files are walked in lexical order, which makes the output deterministic
// but means that for very large directories WalkDir can be inefficient.
// WalkDir does not follow symbolic links.
func (a Afero) WalkDir(root string, fn iofs.WalkDirFunc) error {
	return WalkDir(a.Fs, root, fn)
}

// WalkDir walks the file tree rooted at root, calling fn for each file or
// directory in the tree, including root. See (Afero).WalkDir for details.
func WalkDir(fs Fs, root string, fn iofs.WalkDirFunc) error {
	info, err := lstatIfPossible(fs, root)
	if err != nil {
		err = fn(root, nil, err)
	} else {
		err = walkDir(fs, root, common.FileInfoDirEntry{FileInfo: info}, fn)
	}

	if err == filepath.SkipDir || err == filepath.SkipAll {
		return nil
	}

	return err
}
