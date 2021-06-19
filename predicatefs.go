package afero

import (
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// PredicateFs filters files (not directories) by predicate,
// which takes file path as an arg.
type PredicateFs struct {
	pred   func(string) bool
	source Fs
}

func NewPredicateFs(source Fs, pred func(string) bool) Fs {
	return &PredicateFs{source: source, pred: pred}
}

type PredicateFile struct {
	f    File
	pred func(string) bool
}

func (p *PredicateFs) validate(path string) error {
	if p.pred(path) {
		return nil
	}
	return syscall.ENOENT
}

func (p *PredicateFs) dirOrValidPath(path string) error {
	dir, err := IsDir(p.source, path)
	if err != nil {
		return err
	}
	if dir {
		return nil
	}
	return p.validate(path)
}

func (p *PredicateFs) Chtimes(path string, a, m time.Time) error {
	if err := p.dirOrValidPath(path); err != nil {
		return err
	}
	return p.source.Chtimes(path, a, m)
}

func (p *PredicateFs) Chmod(path string, mode os.FileMode) error {
	if err := p.dirOrValidPath(path); err != nil {
		return err
	}
	return p.source.Chmod(path, mode)
}

func (p *PredicateFs) Chown(path string, uid, gid int) error {
	if err := p.dirOrValidPath(path); err != nil {
		return err
	}
	return p.source.Chown(path, uid, gid)
}

func (p *PredicateFs) Name() string {
	return "PredicateFs"
}

func (p *PredicateFs) Stat(path string) (os.FileInfo, error) {
	if err := p.dirOrValidPath(path); err != nil {
		return nil, err
	}
	return p.source.Stat(path)
}

func (p *PredicateFs) Rename(oldname, newname string) error {
	dir, err := IsDir(p.source, oldname)
	if err != nil {
		return err
	}
	if dir {
		return nil
	}
	if err := p.validate(oldname); err != nil {
		return err
	}
	if err := p.validate(newname); err != nil {
		return err
	}
	return p.source.Rename(oldname, newname)
}

func (p *PredicateFs) RemoveAll(path string) error {
	dir, err := IsDir(p.source, path)
	if err != nil {
		return err
	}
	if !dir {
		if err := p.validate(path); err != nil {
			return err
		}
	}
	return p.source.RemoveAll(path)
}

func (p *PredicateFs) Remove(path string) error {
	if err := p.dirOrValidPath(path); err != nil {
		return err
	}
	return p.source.Remove(path)
}

func (p *PredicateFs) OpenFile(path string, flag int, perm os.FileMode) (File, error) {
	if err := p.dirOrValidPath(path); err != nil {
		return nil, err
	}
	return p.source.OpenFile(path, flag, perm)
}

func (p *PredicateFs) Open(path string) (File, error) {
	dir, err := IsDir(p.source, path)
	if err != nil {
		return nil, err
	}
	if !dir {
		if err := p.validate(path); err != nil {
			return nil, err
		}
	}
	f, err := p.source.Open(path)
	if err != nil {
		return nil, err
	}
	return &PredicateFile{f: f, pred: p.pred}, nil
}

func (p *PredicateFs) Mkdir(n string, path os.FileMode) error {
	return p.source.Mkdir(n, path)
}

func (p *PredicateFs) MkdirAll(n string, path os.FileMode) error {
	return p.source.MkdirAll(n, path)
}

func (p *PredicateFs) Create(path string) (File, error) {
	if err := p.validate(path); err != nil {
		return nil, err
	}
	return p.source.Create(path)
}

func (f *PredicateFile) Close() error {
	return f.f.Close()
}

func (f *PredicateFile) Read(s []byte) (int, error) {
	return f.f.Read(s)
}

func (f *PredicateFile) ReadAt(s []byte, o int64) (int, error) {
	return f.f.ReadAt(s, o)
}

func (f *PredicateFile) Seek(o int64, w int) (int64, error) {
	return f.f.Seek(o, w)
}

func (f *PredicateFile) Write(s []byte) (int, error) {
	return f.f.Write(s)
}

func (f *PredicateFile) WriteAt(s []byte, o int64) (int, error) {
	return f.f.WriteAt(s, o)
}

func (f *PredicateFile) Name() string {
	return f.f.Name()
}

func (f *PredicateFile) Readdir(c int) (fi []os.FileInfo, err error) {
	var pfi []os.FileInfo
	pfi, err = f.f.Readdir(c)
	if err != nil {
		return nil, err
	}
	for _, i := range pfi {
		if i.IsDir() || f.pred(filepath.Join(f.f.Name(), i.Name())) {
			fi = append(fi, i)
		}
	}
	return fi, nil
}

func (f *PredicateFile) Readdirnames(c int) (n []string, err error) {
	fi, err := f.Readdir(c)
	if err != nil {
		return nil, err
	}
	for _, s := range fi {
		n = append(n, s.Name())
	}
	return n, nil
}

func (f *PredicateFile) Stat() (os.FileInfo, error) {
	return f.f.Stat()
}

func (f *PredicateFile) Sync() error {
	return f.f.Sync()
}

func (f *PredicateFile) Truncate(s int64) error {
	return f.f.Truncate(s)
}

func (f *PredicateFile) WriteString(s string) (int, error) {
	return f.f.WriteString(s)
}
