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

// Symlinker is an optional interface in Afero. It is only implemented by the
// filesystems saying so.
type Symlinker interface {
    // EvalSymlinksIfPossible evaluates the symbolic links in the given path.
    // It will EvalSymlinks if the filesystem itself is, or it delegates to, the os filesystem.
    // Else it will call Stat.
    // In addition to the resolved path, it will return a boolean telling whether EvalSymlinks was called or not.
    EvalSymlinksIfPossible(path string) (string, bool, error)

    // SymlinkIfPossible creates a symbolic link from oldname to newname.
    // It will Symlink if the filesystem itself is, or it delegates to, the os filesystem.
    // Else it will call do nothing.
    // It will also return a boolean telling whether Symlink was called or not.
    SymlinkIfPossible(oldname, newname string) (bool, error)
}
