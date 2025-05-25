package afero

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
)

var ErrInvalidRoot = errors.New("invalid root")

// *rootFs implements Root
var _ Root = (*rootFs)(nil)

// Root is a filesystem constrained to a particular directory.
// see https://go.dev/blog/osroot
type Root interface {
	Name() string
	FS() Fs
	OpenRoot(string) (Root, error)
	Close() error
	Create(string) (File, error)
	Lstat(name string) (fs.FileInfo, error)
	Mkdir(name string, perm fs.FileMode) error
	Open(name string) (File, error)
	OpenFile(name string, flag int, perm fs.FileMode) (File, error)
}

// a *rootFs is an Fs that can satisfy [Root], leveraging *BasePathFs for most work
type rootFs struct {
	*BasePathFs
}

// NewRootFs creates a *rootFs rooted at subDir
func NewRootFs(fileSystem Fs, subDir string) (*rootFs, error) {
	info, err := fileSystem.Stat(subDir)
	if err != nil {
		return nil, fmt.Errorf("%w. %w", ErrInvalidRoot, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w. not a directory", ErrInvalidRoot)
	}

	b := &BasePathFs{
		source: fileSystem,
		path:   subDir,
	}
	return &rootFs{b}, nil
}

// Name returns the directory that this *rootedFs is rooted at
func (r *rootFs) Name() string {
	if r.isClosed() {
		return ""
	}
	return r.path
}

// Open opens a file with a root-relative path
func (r *rootFs) Open(name string) (File, error) {
	if r.isClosed() {
		return nil, fmt.Errorf("%w. already closed", ErrInvalidRoot)
	}
	if !filepath.IsLocal(name) {
		return nil, fmt.Errorf("%w. %s is not local", ErrInvalidRoot, name)
	}
	parentFile, err := r.BasePathFs.Open(name)
	if err != nil {
		return nil, err
	}
	rootedFile := &BasePathFile{parentFile, r.path}
	return rootedFile, nil
}

// a *rootFs is closed if it was never opened or it Close() was called
func (r *rootFs) isClosed() bool {
	return (r.BasePathFs == nil)
}

// Close closes a Root. It is an error to use it after this.
func (r *rootFs) Close() error {
	if r.isClosed() {
		return fmt.Errorf("%w. already closed", ErrInvalidRoot)
	}
	r.BasePathFs = nil
	return nil
}

func (r *rootFs) FS() Fs {
	if r.isClosed() {
		return nil
	}
	// *rootedFs already is an [Fs], so just return it
	return r
}

func (r *rootFs) Lstat(name string) (fs.FileInfo, error) {
	if r.isClosed() {
		return nil, ErrInvalidRoot
	}
	info, _, err := r.LstatIfPossible(name)
	return info, err
}

// OpenRoot opens a [Root] inside a *rootFs
func (r *rootFs) OpenRoot(name string) (Root, error) {
	if r.isClosed() {
		return nil, ErrInvalidRoot
	}
	//	the new root is the old root with a new path
	subFs, err := NewRootFs(r.source, filepath.Join(r.path, name))
	if err != nil {
		return nil, fmt.Errorf("could not open root. %w", err)
	}
	return subFs, nil
}
