package afero

import (
	"os"
	"time"
)

// An afero Fs with an extra filter
//
// The FilterFs is run before the source Fs, any non nil error is returned
// to the caller without going to the source Fs. If every filter in the
// chain returns a nil error, the call is sent to the source Fs.
//
// see the TestReadonlyRemoveAndRead() in filter_test.go for an example use
// of filtering (e.g. admins get write access, normal users just readonly)
type FilterFs interface {
	Fs
	AddFilter(FilterFs)
	SetSource(Fs)
}

type Filter struct {
	source Fs
}

func (f *Filter) SetSource(fs Fs) {
	f.source = fs
}

// create a new FilterFs that implements Fs, argument must be an Fs, not
// a FilterFs
func NewFilter(fs Fs) FilterFs {
	return &Filter{source: fs}
}

// prepend a filter in the filter chain
func (f *Filter) AddFilter(fs FilterFs) {
	fs.SetSource(f.source)
	f.source = fs
}

func (f *Filter) Create(name string) (file File, err error) {
	return f.source.Create(name)
}

func (f *Filter) Mkdir(name string, perm os.FileMode) (error) {
	return f.source.Mkdir(name, perm)
}

func (f *Filter) MkdirAll(path string, perm os.FileMode) (error) {
	return f.source.MkdirAll(path, perm)
}

func (f *Filter) Open(name string) (File, error) {
	return f.source.Open(name)
}

func (f *Filter) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	return f.source.OpenFile(name, flag, perm)
}

func (f *Filter) Remove(name string) (error) {
	return f.source.Remove(name)
}

func (f *Filter) RemoveAll(path string) (error) {
	return f.source.RemoveAll(path)
}

func (f *Filter) Rename(oldname, newname string) (error) {
	return f.source.Rename(oldname, newname)
}

func (f *Filter) Stat(name string) (os.FileInfo, error) {
	return f.source.Stat(name)
}

func (f *Filter) Name() string {
	return f.source.Name()
}

func (f *Filter) Chmod(name string, mode os.FileMode) (error) {
	return f.source.Chmod(name, mode)
}

func (f *Filter) Chtimes(name string, atime, mtime time.Time) (error) {
	return f.source.Chtimes(name, atime, mtime)
}
