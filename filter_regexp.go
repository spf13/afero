package afero

import (
	"os"
	"regexp"
	"syscall"
	"time"
)

type RegexpFilter struct {
	file *regexp.Regexp
	dir  *regexp.Regexp
}

func NewRegexpFilter(file *regexp.Regexp, dir *regexp.Regexp) Fs {
	return &RegexpFilter{file: file, dir: dir}
}

func (r *RegexpFilter) Chtimes(n string, a, m time.Time) error {
	if !r.file.MatchString(n) {
		return syscall.ENOENT
	}
	return nil
}

func (r *RegexpFilter) Chmod(n string, m os.FileMode) error {
	if !r.file.MatchString(n) {
		return syscall.ENOENT
	}
	return nil
}

func (r *RegexpFilter) Name() string {
	return "RegexpFilter"
}

func (r *RegexpFilter) Stat(n string) (os.FileInfo, error) {
	// FIXME - what about Stat() on dirs?
	if !r.file.MatchString(n) {
		return nil, syscall.ENOENT
	}
	return nil, nil
}

func (r *RegexpFilter) Rename(o, n string) error {
	// FIXME - what about renaming dirs?
	switch {
	case !r.file.MatchString(o):
		return syscall.ENOENT
	case !r.file.MatchString(n):
		return syscall.EPERM
	default:
		return nil
	}
}

func (r *RegexpFilter) RemoveAll(p string) error {
	if !r.dir.MatchString(p) {
		return syscall.EPERM // FIXME ENOENT?
	}
	return nil
}

func (r *RegexpFilter) Remove(n string) error {
	if !r.file.MatchString(n) {
		return syscall.ENOENT
	}
	return nil
}

func (r *RegexpFilter) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	if !r.file.MatchString(name) {
		return nil, syscall.ENOENT
	}
	return nil, nil
}

func (r *RegexpFilter) Open(n string) (File, error) {
	if !r.file.MatchString(n) {
		return nil, syscall.ENOENT
	}
	return nil, nil
}

func (r *RegexpFilter) Mkdir(n string, p os.FileMode) error {
	if !r.dir.MatchString(n) {
		return syscall.EPERM
	}
	return nil
}

func (r *RegexpFilter) MkdirAll(n string, p os.FileMode) error {
	if !r.dir.MatchString(n) {
		return syscall.EPERM
	}
	return nil
}

func (r *RegexpFilter) Create(n string) (File, error) {
	if !r.file.MatchString(n) {
		return nil, syscall.EPERM
	}
	return nil, nil
}
