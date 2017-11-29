// Copyright © 2017 Blake Williams <code@shabbyrobe.org>
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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MountableFs allows different paths in a hierarchy to be served by different
// afero.Fs objects.
type MountableFs struct {
	node *mountableNode

	// If true, it is possible to mount an Fs over an existing file or directory.
	// If false, attempting to do so will result in an error.
	AllowMasking bool

	// If true, the same Fs can be mounted inside an existing mount of the same Fs,
	// for e.g:
	//	child := afero.NewMemMapFs()
	//	mfs.Mount("/yep", child)
	//	mfs.Mount("/yep/yep", child)
	AllowRecursiveMount bool

	now func() time.Time
}

func NewMountableFs(base Fs) *MountableFs {
	if base == nil {
		base = NewMemMapFs()
	}
	mfs := &MountableFs{
		now:  time.Now,
		node: &mountableNode{fs: base, nodes: map[string]*mountableNode{}},
	}
	return mfs
}

// Mount an afero.Fs at the specified path.
//
// This will fail if there is already a Fs at the path, or
// any existing mounted Fs contains a file at that path.
//
// You must wrap an afero.OsFs in an afero.BasePathFs to mount it,
// even if that's just to dispose of the Windows drive letter.
func (m *MountableFs) Mount(path string, fs Fs) error {
	// No idea what to do with windows drive letters here, so force BasePathFs
	if _, ok := fs.(*OsFs); ok {
		return errOsFs
	}

	if info, err := m.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		if !m.AllowMasking && info != nil && !IsMountNode(info) {
			return os.ErrExist
		}
	}

	parts := splitPath(path)

	cur := m.node
	for i, p := range parts {
		var next *mountableNode
		var ok bool
		if next, ok = cur.nodes[p]; !ok {
			next = &mountableNode{
				nodes:   map[string]*mountableNode{},
				parent:  cur,
				name:    p,
				depth:   i + 1,
				modTime: m.now()}
		}
		if next.fs == fs && !m.AllowRecursiveMount {
			return errRecursiveMount
		}
		cur.nodes[p] = next
		cur = next
	}
	if cur.fs != nil {
		return errAlreadyMounted
	}
	if cur.parent != nil {
		cur.parent.mountedNodes++
	}

	cur.fs = fs
	return nil
}

func (m *MountableFs) Umount(path string) error {
	parts := splitPath(path)

	cur := m.node
	for _, p := range parts {
		if next, ok := cur.nodes[p]; ok {
			cur = next
		} else {
			return &os.PathError{Err: errNotMounted, Op: "Umount", Path: path}
		}
	}
	if cur.fs == nil {
		return &os.PathError{Err: errNotMounted, Op: "Umount", Path: path}
	}

	for cur != nil {
		// Don't stuff around with the root node!
		if cur.parent != nil {
			cur.fs = nil
			cur.parent.mountedNodes--
			if len(cur.nodes) == 0 {
				delete(cur.parent.nodes, cur.name)
			}
		}
		cur = cur.parent
	}

	return nil
}

func (m *MountableFs) Remount(path string, fs Fs) error {
	if err := m.Umount(path); err != nil {
		return wrapErrorPath(path, err)
	}
	return m.Mount(path, fs)
}

func (m *MountableFs) Mkdir(name string, perm os.FileMode) error {
	node := m.node.findNode(name)
	if node != nil {
		// if the path points to an intermediate node and the intermediate node
		// doesn't mask a real directory on the underlying filesystem,
		// make the directory inside the parent filesystem.
		if exists, err := m.reallyExists(name); err != nil || exists {
			return wrapErrorPath(name, err)
		}
		fsNode := node.parentWithFs()
		if fsNode == nil {
			return &os.PathError{Err: os.ErrNotExist, Op: "Mkdir", Path: name}
		}
		rel, err := filepath.Rel(fsNode.fullName(), name)
		if err != nil {
			return wrapErrorPath(name, err)
		}
		rel = string(filepath.Separator) + rel
		if err := fsNode.fs.Mkdir(rel, perm); err != nil {
			return wrapErrorPath(name, err)
		}
		return nil

	} else {
		fs, _, rel := m.node.findPath(name)
		err := wrapErrorPath(name, fs.Mkdir(rel, perm))
		return err
	}
}

func (m *MountableFs) MkdirAll(path string, perm os.FileMode) error {
	parts := splitPath(path)
	partlen := len(parts)
	for i := 0; i <= partlen; i++ {
		cur := joinPath(parts[0:i])
		if err := m.Mkdir(cur, perm); err != nil && !os.IsExist(err) {
			return err
		}
	}
	return nil
}

func (m *MountableFs) Create(name string) (File, error) {
	return m.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

func (m *MountableFs) Open(name string) (File, error) {
	return m.OpenFile(name, os.O_RDONLY, 0)
}

func (m *MountableFs) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	fs, _, rel := m.node.findPath(name)

	exists := true
	isdir, err := IsDir(fs, rel)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, wrapErrorPath(name, err)
		} else {
			exists = false
		}
	}

	if isdir || !exists {
		node := m.node.findNode(name)
		if node != nil {
			file, err := fs.OpenFile(rel, flag, perm)
			if err != nil && !os.IsNotExist(err) {
				return nil, wrapErrorPath(name, err)
			}
			mf := &mountableFile{file: file, node: node, base: rel, name: node.name}
			return mf, nil
		}
	}

	// if we try to write a file into an intermediate node not backed by a
	// directory, we should create it to preserve their illusion:
	if flag&os.O_CREATE == os.O_CREATE {
		parentName := filepath.Dir(name)
		parent := m.node.findNode(parentName)

		if parent != nil && parent.fs == nil {
			parts := splitPath(parentName)
			i := len(parts)

			var fs Fs
			next := parent
			for next != nil && fs == nil {
				if next.fs != nil {
					fs = next.fs
				}
				if next.parent != nil {
					i--
				}
				next = next.parent
			}
			for j := range parts[i:] {
				if err := fs.Mkdir(joinPath(parts[i:i+j+1]), perm|0111); err != nil && !os.IsExist(err) {
					return nil, wrapErrorPath(name, err)
				}
			}
		}
	}

	return fs.OpenFile(rel, flag, perm)
}

func (m *MountableFs) Remove(name string) error {
	fs, _, rel := m.node.findPath(name)
	return wrapErrorPath(name, fs.Remove(rel))
}

func (m *MountableFs) RemoveAll(path string) error {
	info, err := lstatIfOs(m, path)
	if err != nil {
		return wrapErrorPath(path, err)
	}
	err = departWalk(m, path, info, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if IsMountNode(info) {
			return nil
		} else {
			if info.IsDir() {
				node := m.node.findNode(path)
				if node != nil {
					return nil
				}
			}
			return m.Remove(path)
		}
	})
	return wrapErrorPath(path, err)
}

func (m *MountableFs) Rename(oldname string, newname string) error {
	ofs, _, orel := m.node.findPath(oldname)
	nfs, _, nrel := m.node.findPath(newname)

	if ofs == nfs {
		return wrapErrorPath(oldname, ofs.Rename(orel, nrel))
	} else {
		return errCrossFsRename
	}
}

func (m *MountableFs) Stat(name string) (os.FileInfo, error) {
	node := m.node.findNode(name)
	if node != nil && node != m.node {
		return mountedDirFromNode(node)
	}
	fs, _, rel := m.node.findPath(name)
	info, err := fs.Stat(rel)
	if err != nil {
		return nil, wrapErrorPath(name, err)
	}
	return info, nil
}

func (m *MountableFs) Name() string {
	return "MountableFs"
}

func (m *MountableFs) Chmod(name string, mode os.FileMode) error {
	fs, _, rel := m.node.findPath(name)
	return wrapErrorPath(name, fs.Chmod(rel, mode))
}

func (m *MountableFs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	fs, _, rel := m.node.findPath(name)
	ok, err := Exists(fs, rel)
	if err != nil {
		return wrapErrorPath(name, err)
	}
	if !ok {
		node := m.node.findNode(name)
		if node == nil {
			return &os.PathError{Err: os.ErrNotExist, Op: "Chtimes", Path: name}
		}
		node.modTime = mtime
		return nil
	} else {
		return wrapErrorPath(name, fs.Chtimes(rel, atime, mtime))
	}
}

// reallyExists returns true if the file or directory exists on the
// base fs or any of the mounted fs, but not if the path is an intermediate
// mounted node (i.e. if you mount a path but the in-between directories don't
// exist).
func (m *MountableFs) reallyExists(name string) (bool, error) {
	s, err := m.Stat(name)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	} else if IsMountNode(s) {
		return false, nil
	}
	return true, nil
}

func wrapErrorPath(path string, err error) error {
	if err == nil {
		return nil
	}
	switch err := err.(type) {
	case *os.PathError:
		err.Path = path
	}
	return err
}

type mountableNode struct {
	fs           Fs
	parent       *mountableNode
	nodes        map[string]*mountableNode
	name         string
	mountedNodes int
	modTime      time.Time
	depth        int
}

func (n *mountableNode) parentWithFs() (node *mountableNode) {
	node = n.parent
	for node != nil {
		if node.fs != nil {
			return
		}
		node = node.parent
	}
	return
}

func (n *mountableNode) fullName() string {
	out := []string{}
	cur := n
	for cur != nil {
		if cur.name != "" {
			out = append([]string{cur.name}, out...)
		}
		cur = cur.parent
	}
	return joinPath(out)
}

func (n *mountableNode) findNode(path string) *mountableNode {
	parts := splitPath(path)
	cur := n
	for _, p := range parts {
		if next, ok := cur.nodes[p]; ok && next != nil {
			cur = next
		} else {
			return nil
		}
	}
	return cur
}

func (n *mountableNode) findPath(path string) (fs Fs, base, rel string) {
	parts := splitPath(path)

	var out Fs
	outIdx := -1
	out = n.fs
	cur := n
	for i, p := range parts {
		if next, ok := cur.nodes[p]; ok {
			cur = next
			if cur.fs != nil {
				out = cur.fs
				outIdx = i
			}
		} else {
			break
		}
	}

	// afero is a bit fussy and unpredictable about leading slashes.
	return out,
		string(filepath.Separator) + filepath.Join(parts[:outIdx+1]...),
		string(filepath.Separator) + filepath.Join(parts[outIdx+1:]...)
}

type mountableFile struct {
	name string
	file File
	node *mountableNode
	base string
}

func (m *mountableFile) Readdir(count int) (out []os.FileInfo, err error) {
	if m.file != nil {
		out, err = m.file.Readdir(count)
		if err != nil {
			return
		}
	}
	if m.node != nil {
		for _, node := range m.node.nodes {
			var mdi *mountedDirInfo
			mdi, err = mountedDirFromNode(node)
			if err != nil {
				return
			}
			out = append(out, mdi)
		}
	}
	return
}

func (m *mountableFile) Readdirnames(n int) (out []string, err error) {
	if m.file != nil {
		out, err = m.file.Readdirnames(n)
		if err != nil {
			return
		}
	}
	if m.node != nil {
		for part := range m.node.nodes {
			out = append(out, part)
		}
	}
	return
}

func (m *mountableFile) Close() error {
	if m.file != nil {
		return m.file.Close()
	}
	return nil
}

func (m *mountableFile) Read(p []byte) (n int, err error) {
	if m.file != nil {
		return m.file.Read(p)
	}
	return 0, errNotAFile
}

func (m *mountableFile) ReadAt(p []byte, off int64) (n int, err error) {
	if m.file != nil {
		return m.file.ReadAt(p, off)
	}
	return 0, errNotAFile
}

func (m *mountableFile) Seek(offset int64, whence int) (int64, error) {
	if m.file != nil {
		return m.file.Seek(offset, whence)
	}
	return 0, errNotAFile
}

func (m *mountableFile) Write(p []byte) (n int, err error) {
	if m.file != nil {
		return m.file.Write(p)
	}
	return 0, errNotAFile
}

func (m *mountableFile) WriteAt(p []byte, off int64) (n int, err error) {
	if m.file != nil {
		return m.file.WriteAt(p, off)
	}
	return 0, errNotAFile
}

func (m *mountableFile) Name() string { return m.name }

func (m *mountableFile) Stat() (os.FileInfo, error) {
	if m.file != nil {
		return m.file.Stat()
	} else {
		if m.node != nil {
			mdi, err := mountedDirFromNode(m.node)
			if err != nil {
				return nil, err
			}
			return mdi, nil
		}
	}
	return nil, os.ErrNotExist
}

func (m *mountableFile) Sync() error {
	if m.file != nil {
		return m.file.Sync()
	}
	return errNotAFile
}

func (m *mountableFile) Truncate(size int64) error {
	if m.file != nil {
		return m.file.Truncate(size)
	}
	return errNotAFile
}

func (m *mountableFile) WriteString(s string) (ret int, err error) {
	if m.file != nil {
		return m.file.WriteString(s)
	}
	return 0, errNotAFile
}

type mountedDirInfo struct {
	name    string
	mode    os.FileMode
	modTime time.Time
}

func (m *mountedDirInfo) Name() string       { return m.name }
func (m *mountedDirInfo) Mode() os.FileMode  { return m.mode | os.ModeDir }
func (m *mountedDirInfo) ModTime() time.Time { return m.modTime }
func (m *mountedDirInfo) IsDir() bool        { return true }
func (m *mountedDirInfo) Sys() interface{}   { return nil }

func (m *mountedDirInfo) Size() int64 {
	// copied from afero, not sure why it's 42.
	return int64(42)
}

func mountedDirFromNode(node *mountableNode) (*mountedDirInfo, error) {
	if node.name == "" {
		panic("missing name from node")
	}
	mdi := &mountedDirInfo{
		name:    node.name,
		mode:    0777,
		modTime: node.modTime,
	}
	if node.fs != nil {
		// dir should inherit stat info of mounted fs root node
		info, err := node.fs.Stat("/")
		if err != nil {
			return nil, err
		}
		mdi.modTime = info.ModTime()
		mdi.mode = info.Mode()
	}
	return mdi, nil
}

var (
	errCrossFsRename  = errors.New("cross-fs rename")
	errRecursiveMount = errors.New("recursive mount")
	errShortCopy      = errors.New("short copy")
	errAlreadyMounted = errors.New("already mounted")
	errNotMounted     = errors.New("not mounted")
	errNotAFile       = errors.New("not a file")
	errOsFs           = errors.New("afero.OsFs should not be mounted - use afero.BasePathFs instead")
)

func underlyingError(err error) error {
	switch err := err.(type) {
	case *os.PathError:
		return err.Err
	}
	return err
}

func IsErrCrossFsRename(err error) bool  { return underlyingError(err) == errCrossFsRename }
func IsErrRecursiveMount(err error) bool { return underlyingError(err) == errRecursiveMount }
func IsErrShortCopy(err error) bool      { return underlyingError(err) == errShortCopy }
func IsErrAlreadyMounted(err error) bool { return underlyingError(err) == errAlreadyMounted }
func IsErrNotMounted(err error) bool     { return underlyingError(err) == errNotMounted }
func IsErrNotAFile(err error) bool       { return underlyingError(err) == errNotAFile }
func IsErrOsFs(err error) bool           { return underlyingError(err) == errOsFs }

func splitPath(path string) []string {
	in := strings.Trim(path, string(filepath.Separator))
	if in == "" {
		return nil
	}
	return strings.Split(in, string(filepath.Separator))
}

func joinPath(parts []string) string {
	return string(filepath.Separator) + strings.Join(parts, string(filepath.Separator))
}

func IsMountNode(info os.FileInfo) bool {
	if _, ok := info.(*mountedDirInfo); ok {
		return true
	}
	return false
}

// departWalk recursively descends path, calling walkFn.
// it calls walkFn on departure rather than arrival, allowing removal
// adapted from afero.walk
func departWalk(fs Fs, path string, info os.FileInfo, walkFn filepath.WalkFunc) error {
	if info.IsDir() {
		names, err := readDirNames(fs, path)
		if err != nil {
			return walkFn(path, info, err)
		}

		for _, name := range names {
			filename := filepath.Join(path, name)
			fileInfo, err := lstatIfOs(fs, filename)
			if err != nil {
				if err := walkFn(filename, fileInfo, err); err != nil {
					return err
				}
			} else {
				err = departWalk(fs, filename, fileInfo, walkFn)
				if err != nil {
					return err
				}
			}
		}
	}
	return walkFn(path, info, nil)
}
