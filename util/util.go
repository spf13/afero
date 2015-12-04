// Copyright ©2015 Steve Francia <spf@spf13.com>
// Portions Copyright ©2015 The Hugo Authors
//
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

package util

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/spf13/afero"
)

type AferoUtilFS struct {
	fs afero.Fs
}

// Filepath separator defined by os.Separator.
const FilePathSeparator = string(filepath.Separator)

// Takes a reader and a path and writes the content
func (a AferoUtilFS) WriteReader(path string, r io.Reader) (err error) {
	return WriteReader(path, r, a.fs)
}

func WriteReader(path string, r io.Reader, fs afero.Fs) (err error) {
	dir, _ := filepath.Split(path)
	ospath := filepath.FromSlash(dir)

	if ospath != "" {
		err = fs.MkdirAll(ospath, 0777) // rwx, rw, r
		if err != nil {
			if err != os.ErrExist {
				log.Fatalln(err)
			}
		}
	}

	file, err := fs.Create(path)
	if err != nil {
		return
	}
	defer file.Close()

	_, err = io.Copy(file, r)
	return
}

// Same as WriteReader but checks to see if file/directory already exists.
func (a AferoUtilFS) SafeWriteReader(path string, r io.Reader) (err error) {
	return SafeWriteReader(path, r, a.fs)
}

func SafeWriteReader(path string, r io.Reader, fs afero.Fs) (err error) {
	dir, _ := filepath.Split(path)
	ospath := filepath.FromSlash(dir)

	if ospath != "" {
		err = fs.MkdirAll(ospath, 0777) // rwx, rw, r
		if err != nil {
			return
		}
	}

	exists, err := Exists(path, fs)
	if err != nil {
		return
	}
	if exists {
		return fmt.Errorf("%v already exists", path)
	}

	file, err := fs.Create(path)
	if err != nil {
		return
	}
	defer file.Close()

	_, err = io.Copy(file, r)
	return
}

func (a AferoUtilFS) GetTempDir(subPath string) string {
	return GetTempDir(subPath, a.fs)
}

// GetTempDir returns the default temp directory with trailing slash
// if subPath is not empty then it will be created recursively with mode 777 rwx rwx rwx
func GetTempDir(subPath string, fs afero.Fs) string {
	addSlash := func(p string) string {
		if FilePathSeparator != p[len(p)-1:] {
			p = p + FilePathSeparator
		}
		return p
	}
	dir := addSlash(os.TempDir())

	if subPath != "" {
		// preserve windows backslash :-(
		if FilePathSeparator == "\\" {
			subPath = strings.Replace(subPath, "\\", "____", -1)
		}
		dir = dir + UnicodeSanitize((subPath))
		if FilePathSeparator == "\\" {
			dir = strings.Replace(dir, "____", "\\", -1)
		}

		if exists, _ := Exists(dir, fs); exists {
			return addSlash(dir)
		}

		err := fs.MkdirAll(dir, 0777)
		if err != nil {
			panic(err)
		}
		dir = addSlash(dir)
	}
	return dir
}

// Rewrite string to remove non-standard path characters
func UnicodeSanitize(s string) string {
	source := []rune(s)
	target := make([]rune, 0, len(source))

	for _, r := range source {
		if unicode.IsLetter(r) ||
			unicode.IsDigit(r) ||
			unicode.IsMark(r) ||
			r == '.' ||
			r == '/' ||
			r == '\\' ||
			r == '_' ||
			r == '-' ||
			r == '%' ||
			r == ' ' ||
			r == '#' {
			target = append(target, r)
		}
	}

	return string(target)
}

// Transform characters with accents into plan forms
func NeuterAccents(s string) string {
	t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
	result, _, _ := transform.String(t, string(s))

	return result
}

func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
}

func (a AferoUtilFS) FileContainsBytes(filename string, subslice []byte) (bool, error) {
	return FileContainsBytes(filename, subslice, a.fs)
}

// Check if a file contains a specified string.
func FileContainsBytes(filename string, subslice []byte, fs afero.Fs) (bool, error) {
	f, err := fs.Open(filename)
	if err != nil {
		return false, err
	}
	defer f.Close()

	return readerContains(f, subslice), nil
}

// readerContains reports whether subslice is within r.
func readerContains(r io.Reader, subslice []byte) bool {

	if r == nil || len(subslice) == 0 {
		return false
	}

	bufflen := len(subslice) * 4
	halflen := bufflen / 2
	buff := make([]byte, bufflen)
	var err error
	var n, i int

	for {
		i++
		if i == 1 {
			n, err = io.ReadAtLeast(r, buff[:halflen], halflen)
		} else {
			if i != 2 {
				// shift left to catch overlapping matches
				copy(buff[:], buff[halflen:])
			}
			n, err = io.ReadAtLeast(r, buff[halflen:], halflen)
		}

		if n > 0 && bytes.Contains(buff, subslice) {
			return true
		}

		if err != nil {
			break
		}
	}
	return false
}

func (a AferoUtilFS) DirExists(path string) (bool, error) {
	return DirExists(path, a.fs)
}

// DirExists checks if a path exists and is a directory.
func DirExists(path string, fs afero.Fs) (bool, error) {
	fi, err := fs.Stat(path)
	if err == nil && fi.IsDir() {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (a AferoUtilFS) IsDir(path string) (bool, error) {
	return IsDir(path, a.fs)
}

// IsDir checks if a given path is a directory.
func IsDir(path string, fs afero.Fs) (bool, error) {
	fi, err := fs.Stat(path)
	if err != nil {
		return false, err
	}
	return fi.IsDir(), nil
}

func (a AferoUtilFS) IsEmpty(path string) (bool, error) {
	return IsEmpty(path, a.fs)
}

// IsEmpty checks if a given file or directory is empty.
func IsEmpty(path string, fs afero.Fs) (bool, error) {
	if b, _ := Exists(path, fs); !b {
		return false, fmt.Errorf("%q path does not exist", path)
	}
	fi, err := fs.Stat(path)
	if err != nil {
		return false, err
	}
	if fi.IsDir() {
		f, err := fs.Open(path)
		defer f.Close()
		if err != nil {
			return false, err
		}
		list, err := f.Readdir(-1)
		return len(list) == 0, nil
	}
	return fi.Size() == 0, nil
}

func (a AferoUtilFS) Exists(path string) (bool, error) {
	return Exists(path, a.fs)
}

// Check if a file or directory exists.
func Exists(path string, fs afero.Fs) (bool, error) {
	_, err := fs.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
