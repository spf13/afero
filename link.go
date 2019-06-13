// Copyright © 2018 Steve Francia <spf@spf13.com>.
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
	"os"
	"runtime"
	"strings"
	"unicode"
)

// Lstater is an optional interface in Afero. It is only implemented by the
// filesystems saying so.
// It will call Lstat if the filesystem iself is, or it delegates to, the os filesystem.
// Else it will call Stat.
// In addtion to the FileInfo, it will return a boolean telling whether Lstat was called or not.
type Lstater interface {
	LstatIfPossible(name string) (os.FileInfo, bool, error)
}

// Like Lstater, Symlinker is an optional interface in Afero - only implemented by
// the filesystems saying so.
// It will call Symlink if the filesystem supports it, otherwise will return false.
type Symlinker interface {
	SymlinkIfPossible(oldname, newname string) (bool, error)
}

// Like Lstater, Readlinker is an optional interface in Afero - only implemented by
// the filesystems saying so.
// It will call Readlink if the filesystem supports it, otherwise will return false.
type Readlinker interface {
	ReadlinkIfPossible(name string) (string, bool, error)
}

func EvalSymlinks(fs Fs, path string) (string, bool, error) {
	//what do we do differently for windows, and when do we know to do it? only
	//for osfs and only when runtime.goos=="windows"?

	lstater, ok := fs.(Lstater)
	if !ok {
		return "", false, nil
	}
	readlinker, ok := fs.(Readlinker)
	if !ok {
		return "", false, nil
	}

	//start with last element and work back
	idx := len(path)
	iterations := 0
	for iterations < 1024 /*arbitrary*/ {
		iterations++
		fi, b, err := lstater.LstatIfPossible(path[:idx])
		if err != nil || b == false {
			return "", b, err
		}
		if fi.Mode()&os.ModeSymlink > 0 {
			//is symlink
			link, b, err := readlinker.ReadlinkIfPossible(path[:idx])
			if err != nil || b == false {
				return "", b, err
			}
			if isAbsolute(link) {
				tailLen := len(path[idx:])
				path = link + path[idx:]
				idx = len(path) - tailLen
				continue
			}
			var tail string
			if idx < len(path) /* not last segment */ {
				tail = path[idx:]
			}
			path, err = resolveRelative(dir(path[:idx]), link)
			if err != nil {
				return "", true, err
			}
			idx = len(path)
			path = path + tail
			continue
		}
		idx = strings.LastIndexByte(path[:idx], os.PathSeparator)
		if idx <= 0 {
			return path, true, nil
		}
	}
	return "", false, os.ErrInvalid
}

func prevSep(path string) int {
	return strings.LastIndexByte(path, os.PathSeparator)
}
func dir(path string) string {
	s := prevSep(path)
	if s < 0 {
		return ""
	}
	return path[:s]
}

func isAbsolute(path string) bool {
	if runtime.GOOS == "windows" {
		//WARNING this should cover most uses but is not perfect
		if len(path) > 0 && path[0] == '\\' {
			return true
		}
		if len(path) > 2 &&
			unicode.IsLetter(rune(path[0])) &&
			path[1] == ':' &&
			path[2] == '\\' {
			return true
		}
		return false
	}
	if len(path) > 0 && path[0] == '/' {
		return true
	}
	return false
}

func resolveRelative(path, link string) (string, error) {
	if path == "" {
		return link, nil
	}
	sep := string(os.PathSeparator)
	noop := sep + sep        //   foo//bar
	noop2 := sep + "." + sep //   foo/./bar
	prev := sep + ".." + sep //   foo/../bar
	path = path + sep + link
	for strings.Contains(path, noop) {
		path = strings.Replace(path, noop, sep, -1)
	}
	for strings.Contains(path, noop2) {
		path = strings.Replace(path, noop2, sep, -1)
	}
	for strings.Contains(path, prev) {
		p := strings.Index(path, prev)
		s := strings.LastIndex(path[:p], sep)
		if s < 0 {
			return "", os.ErrInvalid
		}
		path = path[:s] + sep + path[p+len(prev):]
	}
	return path, nil
}
