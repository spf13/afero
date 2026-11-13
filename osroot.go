package afero

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var (
	_ Symlinker  = (*OsRootFs)(nil)
	_ FsCloser   = (*OsRootFs)(nil)
	_ RootOpener = (*OsRootFs)(nil)
)

// FsCloser is a Fs that holds resources. Call Close when done.
type FsCloser interface {
	Fs
	io.Closer
}

// RootOpener is an optional interface in Afero. It is only implemented by
// the filesystems that can open a root backed by os.Root.
type RootOpener interface {
	OpenRoot(name string) (FsCloser, error)
}

// NewRootFs returns a Fs limited to path inside source.
//
// If source implements RootOpener, the result uses os.Root, which also blocks
// escapes through symbolic links. Otherwise the result is a BasePathFs,
// which checks paths as text only.
//
// The caller must call Close when done.
func NewRootFs(source Fs, path string) (FsCloser, error) {
	if opener, ok := source.(RootOpener); ok {
		return opener.OpenRoot(path)
	}
	return newBasePathRootFs(source, path)
}

// basePathRootFs is the NewRootFs fallback. Close does nothing.
// It embeds *BasePathFs, not Fs, so Lstater and Symlinker stay visible.
type basePathRootFs struct{ *BasePathFs }

func (basePathRootFs) Close() error { return nil }

// newBasePathRootFs returns a basePathRootFs at path inside source,
// after it makes sure that path is a directory.
func newBasePathRootFs(source Fs, path string) (FsCloser, error) {
	info, err := source.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &os.PathError{Op: "openroot", Path: path, Err: syscall.ENOTDIR}
	}
	return basePathRootFs{&BasePathFs{source: source, path: path}}, nil
}

// OsRootFs is a Fs implementation that uses an *os.Root. All operations
// stay inside the directory of the root.
//
// Names are relative to the root. A leading path separator is ignored,
// so "/a" and "a" refer to the same file, and "/" refers to the root itself.
//
// For details in any method, check the documentation of os.Root
// (https://pkg.go.dev/os#Root).
type OsRootFs struct {
	root *os.Root
	// owned is true if OpenRoot opened root, so Close must close it.
	owned bool
}

// NewOsRootFs returns a Fs that operates inside root.
// The caller is responsible for closing root when it is no longer needed.
func NewOsRootFs(root *os.Root) *OsRootFs {
	return &OsRootFs{root: root}
}

// OpenRoot opens a root at name, inside this root.
func (r *OsRootFs) OpenRoot(name string) (FsCloser, error) {
	root, err := r.root.OpenRoot(rootRelPath(name))
	if err != nil {
		return nil, err
	}
	return &OsRootFs{root: root, owned: true}, nil
}

// Close closes the root if OpenRoot opened it.
// A root passed to NewOsRootFs stays open. Its owner closes it.
func (r *OsRootFs) Close() error {
	if !r.owned {
		return nil
	}
	return r.root.Close()
}

// rootRelPath converts name to the root-relative form that os.Root expects.
func rootRelPath(name string) string {
	name = strings.TrimLeft(filepath.ToSlash(name), "/")
	if name == "" {
		return "."
	}
	return name
}

func (*OsRootFs) Name() string { return "OsRootFs" }

func (r *OsRootFs) Create(name string) (File, error) {
	f, e := r.root.Create(rootRelPath(name))
	if f == nil {
		// while this looks strange, we need to return a bare nil (of type nil) not
		// a nil value of type *os.File or nil won't be nil
		return nil, e
	}
	return f, e
}

func (r *OsRootFs) Mkdir(name string, perm os.FileMode) error {
	return r.root.Mkdir(rootRelPath(name), perm)
}

func (r *OsRootFs) MkdirAll(path string, perm os.FileMode) error {
	return r.root.MkdirAll(rootRelPath(path), perm)
}

func (r *OsRootFs) Open(name string) (File, error) {
	f, e := r.root.Open(rootRelPath(name))
	if f == nil {
		// while this looks strange, we need to return a bare nil (of type nil) not
		// a nil value of type *os.File or nil won't be nil
		return nil, e
	}
	return f, e
}

func (r *OsRootFs) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	f, e := r.root.OpenFile(rootRelPath(name), flag, perm)
	if f == nil {
		// while this looks strange, we need to return a bare nil (of type nil) not
		// a nil value of type *os.File or nil won't be nil
		return nil, e
	}
	return f, e
}

func (r *OsRootFs) Remove(name string) error {
	return r.root.Remove(rootRelPath(name))
}

func (r *OsRootFs) RemoveAll(path string) error {
	return r.root.RemoveAll(rootRelPath(path))
}

func (r *OsRootFs) Rename(oldname, newname string) error {
	return r.root.Rename(rootRelPath(oldname), rootRelPath(newname))
}

func (r *OsRootFs) Stat(name string) (os.FileInfo, error) {
	return r.root.Stat(rootRelPath(name))
}

func (r *OsRootFs) Chmod(name string, mode os.FileMode) error {
	return r.root.Chmod(rootRelPath(name), mode)
}

func (r *OsRootFs) Chown(name string, uid, gid int) error {
	return r.root.Chown(rootRelPath(name), uid, gid)
}

func (r *OsRootFs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return r.root.Chtimes(rootRelPath(name), atime, mtime)
}

func (r *OsRootFs) LstatIfPossible(name string) (os.FileInfo, bool, error) {
	fi, err := r.root.Lstat(rootRelPath(name))
	return fi, true, err
}

// SymlinkIfPossible creates newname as a symbolic link to oldname.
// The link may point outside the root, but the root does not follow it.
func (r *OsRootFs) SymlinkIfPossible(oldname, newname string) error {
	return r.root.Symlink(oldname, rootRelPath(newname))
}

func (r *OsRootFs) ReadlinkIfPossible(name string) (string, error) {
	return r.root.Readlink(rootRelPath(name))
}
