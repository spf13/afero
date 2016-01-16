package afero

import (
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// The CopyOnWriteFs is a union filesystem: a read only base file system with
// a possibly writeable layer on top. Changes to the file system will only
// be made in the overlay: Changing an existing file in the base layer which
// is not present in the overlay will copy the file to the overlay ("changing"
// includes also calls to e.g. Chtimes() and Chmod()).
//
// Reading directories is currently only supported via Open(), not OpenFile().
type CopyOnWriteFs struct {
	base  Fs
	layer Fs
}

func NewCopyOnWriteFs(base Fs, layer Fs) Fs {
	return &CopyOnWriteFs{base: base, layer: layer}
}

// isBaseFile Returns true if the given file is only found in the base layer
// will return true if file is not found in either layer
func (u *CopyOnWriteFs) isBaseFile(name string) (bool, error) {
	if _, err := u.layer.Stat(name); err == nil {
		return false, nil
	}
	_, err := u.base.Stat(name)
	return true, err
}

func (u *CopyOnWriteFs) copyToLayer(name string) error {
	return copyToLayer(u.base, u.layer, name)
}

func (u *CopyOnWriteFs) Chtimes(name string, atime, mtime time.Time) error {
	b, err := u.isBaseFile(name)
	if err != nil {
		return err
	}
	if b {
		if err := u.copyToLayer(name); err != nil {
			return err
		}
	}
	return u.layer.Chtimes(name, atime, mtime)
}

func (u *CopyOnWriteFs) Chmod(name string, mode os.FileMode) error {
	b, err := u.isBaseFile(name)
	if err != nil {
		return err
	}
	if b {
		if err := u.copyToLayer(name); err != nil {
			return err
		}
	}
	return u.layer.Chmod(name, mode)
}

func (u *CopyOnWriteFs) Stat(name string) (os.FileInfo, error) {
	fi, err := u.layer.Stat(name)
	switch err {
	case nil:
		return fi, nil
	case syscall.ENOENT:
		return u.base.Stat(name)
	default:
		return nil, err
	}
}

// Renaming files present only in the base layer is not permitted
func (u *CopyOnWriteFs) Rename(oldname, newname string) error {
	b, err := u.isBaseFile(oldname)
	if err != nil {
		return err
	}
	if b {
		return syscall.EPERM
	}
	return u.layer.Rename(oldname, newname)
}

// Removing files present only in the base layer is not permitted. If
// a file is present in the base layer and the overlay, only the overlay
// will be removed.
func (u *CopyOnWriteFs) Remove(name string) error {
	err := u.layer.Remove(name)
	switch err {
	case syscall.ENOENT:
		_, err = u.base.Stat(name)
		if err == nil {
			return syscall.EPERM
		}
		return syscall.ENOENT
	default:
		return err
	}
}

func (u *CopyOnWriteFs) RemoveAll(name string) error {
	err := u.layer.RemoveAll(name)
	switch err {
	case syscall.ENOENT:
		_, err = u.base.Stat(name)
		if err == nil {
			return syscall.EPERM
		}
		return syscall.ENOENT
	default:
		return err
	}
}

func (u *CopyOnWriteFs) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	b, err := u.isBaseFile(name)
	if err != nil {
		return nil, err
	}

	if flag&(os.O_WRONLY|os.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC) != 0 {
		if b {
			if err = u.copyToLayer(name); err != nil {
				return nil, err
			}
			return u.layer.OpenFile(name, flag, perm)
		}

		dir := filepath.Dir(name)
		isaDir, err := IsDir(u.base, dir)
		if err != nil {
			return nil, err
		}
		if isaDir {
			if err = u.layer.MkdirAll(dir, 0777); err != nil {
				return nil, err
			}
			return u.layer.OpenFile(name, flag, perm)
		}

		isaDir, err = IsDir(u.layer, dir)
		if err != nil {
			return nil, err
		}
		if isaDir {
			return u.layer.OpenFile(name, flag, perm)
		}

		return nil, &os.PathError{Op: "open", Path: name, Err: syscall.ENOTDIR} // ...or os.ErrNotExist?
	}
	if b {
		return u.base.OpenFile(name, flag, perm)
	}
	return u.layer.OpenFile(name, flag, perm)
}

func (u *CopyOnWriteFs) Open(name string) (File, error) {
	b, err := u.isBaseFile(name)
	if err != nil {
		return nil, err
	}

	if b {
		// If it's only in the base (not overlay) return that File
		return u.base.Open(name)
	}

	dir, err := IsDir(u.layer, name)
	if err != nil {
		return nil, err
	}
	if !dir {
		// If it's in the overlay and not a directory, return that file
		return u.layer.Open(name)
	}

	bfile, _ := u.base.Open(name)
	lfile, err := u.layer.Open(name)
	if err != nil && bfile == nil {
		return nil, err
	}
	// If it's a directory in both, return a unionFile
	return &UnionFile{base: bfile, layer: lfile}, nil
}

func (u *CopyOnWriteFs) Mkdir(name string, perm os.FileMode) error {
	dir, err := IsDir(u.base, name)
	if err != nil {
		return u.layer.MkdirAll(name, perm)
	}
	if dir {
		return syscall.EEXIST
	}
	return u.layer.MkdirAll(name, perm)
}

func (u *CopyOnWriteFs) Name() string {
	return "CopyOnWriteFs"
}

func (u *CopyOnWriteFs) MkdirAll(name string, perm os.FileMode) error {
	dir, err := IsDir(u.base, name)
	if err != nil {
		return u.layer.MkdirAll(name, perm)
	}
	if dir {
		return syscall.EEXIST
	}
	return u.layer.MkdirAll(name, perm)
}

func (u *CopyOnWriteFs) Create(name string) (File, error) {
	return u.OpenFile(name, os.O_TRUNC|os.O_RDWR, 0666)
}
