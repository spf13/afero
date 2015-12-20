package afero

import (
	"os"
	"syscall"
	"time"
)

type ReadOnlyFilter struct {
	source Fs
}

func NewReadonlyFilter() FilterFs {
	return &ReadOnlyFilter{}
}

func (r *ReadOnlyFilter) SetSource(fs Fs) {
	r.source = fs
}

// prepend a filter in the filter chain
func (r *ReadOnlyFilter) AddFilter(fs FilterFs) {
	fs.SetSource(r.source)
	r.source = fs
}

func (r *ReadOnlyFilter) ReadDir(name string) ([]os.FileInfo, error) {
	return ReadDir(r.source, name)
}

func (r *ReadOnlyFilter) Chtimes(n string, a, m time.Time) error {
	return syscall.EPERM
}

func (r *ReadOnlyFilter) Chmod(n string, m os.FileMode) error {
	return syscall.EPERM
}

func (r *ReadOnlyFilter) Name() string {
	return "ReadOnlyFilter"
}

func (r *ReadOnlyFilter) Stat(name string) (os.FileInfo, error) {
	return r.source.Stat(name)
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
	return r.source.OpenFile(name, flag, perm)
}

func (r *ReadOnlyFilter) Open(n string) (File, error) {
	return r.source.Open(n)
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
