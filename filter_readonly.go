package afero

import (
	"os"
	"syscall"
	"time"
)

type ReadOnlyFilter struct {
}

func NewReadonlyFilter() Fs {
	return &ReadOnlyFilter{}
}

func (r *ReadOnlyFilter) Chtimes(n string, a, m time.Time) error {
	return syscall.EPERM
}

func (r *ReadOnlyFilter) Chmod(n string, m os.FileMode) error {
	return syscall.EPERM
}

func (r *ReadOnlyFilter) Name() string {
	return "readOnlyFilter"
}

func (r *ReadOnlyFilter) Stat(name string) (os.FileInfo, error) {
	return nil, nil
}

func (r *ReadOnlyFilter) Rename(o, n string) error {
	return syscall.EPERM
}

func (r *ReadOnlyFilter) RemoveAll(p string) error {
	return syscall.EPERM
}

func (r *ReadOnlyFilter) Remove(n string) error {
	return syscall.EPERM
}

func (r *ReadOnlyFilter) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	if flag&(os.O_WRONLY|syscall.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC) != 0 {
		return nil, syscall.EPERM
	}
	return nil, nil
}

func (r *ReadOnlyFilter) Open(n string) (File, error) {
	return nil, nil
}

func (r *ReadOnlyFilter) Mkdir(n string, p os.FileMode) error {
	return syscall.EPERM
}

func (r *ReadOnlyFilter) MkdirAll(n string, p os.FileMode) error {
	return syscall.EPERM
}

func (r *ReadOnlyFilter) Create(n string) (File, error) {
	return nil, syscall.EPERM
}
