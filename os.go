// Copyright © 2014 Steve Francia <spf@spf13.com>.
// Copyright 2013 tsuru authors. All rights reserved.
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
	"time"
)

var _ Lstater = (*OsFs)(nil)

// OsFs is a Fs implementation that uses functions provided by the os package.
//
// For details in any method, check the documentation of the os package
// (http://golang.org/pkg/os/).
type OsFs struct{}

func NewOsFs() Fs {
	return &OsFs{}
}

func (OsFs) Name() string { return "OsFs" }

func (OsFs) Create(name string) (File, error) {
	f, e := os.Create(normalizeLongPath(name))
	if f == nil {
		// while this looks strange, we need to return a bare nil (of type nil) not
		// a nil value of type *os.File or nil won't be nil
		return nil, e
	}
	return f, e
}

func (OsFs) Mkdir(name string, perm os.FileMode) error {
	return os.Mkdir(normalizeLongPath(name), perm)
}

func (OsFs) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(normalizeLongPath(path), perm)
}

func (OsFs) Open(name string) (File, error) {
	f, e := os.Open(normalizeLongPath(name))
	if f == nil {
		// while this looks strange, we need to return a bare nil (of type nil) not
		// a nil value of type *os.File or nil won't be nil
		return nil, e
	}
	return f, e
}

func (OsFs) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	f, e := os.OpenFile(normalizeLongPath(name), flag, perm)
	if f == nil {
		// while this looks strange, we need to return a bare nil (of type nil) not
		// a nil value of type *os.File or nil won't be nil
		return nil, e
	}
	return f, e
}

func (OsFs) Remove(name string) error {
	return os.Remove(normalizeLongPath(name))
}

func (OsFs) RemoveAll(path string) error {
	return os.RemoveAll(normalizeLongPath(path))
}

func (OsFs) Rename(oldname, newname string) error {
	return os.Rename(normalizeLongPath(oldname), normalizeLongPath(newname))
}

func (OsFs) Stat(name string) (os.FileInfo, error) {
	return os.Stat(normalizeLongPath(name))
}

func (OsFs) Chmod(name string, mode os.FileMode) error {
	return os.Chmod(normalizeLongPath(name), mode)
}

func (OsFs) Chown(name string, uid, gid int) error {
	return os.Chown(normalizeLongPath(name), uid, gid)
}

func (OsFs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return os.Chtimes(normalizeLongPath(name), atime, mtime)
}

func (OsFs) LstatIfPossible(name string) (os.FileInfo, bool, error) {
	fi, err := os.Lstat(normalizeLongPath(name))
	return fi, true, err
}

func (OsFs) SymlinkIfPossible(oldname, newname string) error {
	return os.Symlink(oldname, newname)
}

func (OsFs) ReadlinkIfPossible(name string) (string, error) {
	return os.Readlink(name)
}
