package afero

import (
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type predFn func(bool, string) bool

// FilePredicateFs filters files (not directories) by predicate,
// which takes file path as an arg.
type FilePredicateFs struct {
	pred   predFn
	source Fs
}

var (
	_ fs.ReadDirFile = (*PredicateFile)(nil)
)

func NewFilePredicateFs(source Fs, pred predFn) Fs {
	return &FilePredicateFs{source: source, pred: pred}
}

type PredicateFile struct {
	f    File
	pred predFn
}

func (p *FilePredicateFs) validate(path string) error {
	dir, err := IsDir(p.source, path)
	if err != nil {
		return err
	}

	if p.pred(dir, path) {
		return nil
	}
	return syscall.ENOENT
}

func (p *FilePredicateFs) Chtimes(path string, a, m time.Time) error {
	if err := p.validate(path); err != nil {
		return err
	}
	return p.source.Chtimes(path, a, m)
}

func (p *FilePredicateFs) Chmod(path string, mode os.FileMode) error {
	if err := p.validate(path); err != nil {
		return err
	}
	return p.source.Chmod(path, mode)
}

func (p *FilePredicateFs) Chown(path string, uid, gid int) error {
	if err := p.validate(path); err != nil {
		return err
	}
	return p.source.Chown(path, uid, gid)
}

func (p *FilePredicateFs) Name() string {
	return "FilePredicateFs"
}

func (p *FilePredicateFs) Stat(path string) (os.FileInfo, error) {
	if err := p.validate(path); err != nil {
		return nil, err
	}
	return p.source.Stat(path)
}

func (p *FilePredicateFs) Rename(oldname, newname string) error {
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

func (p *FilePredicateFs) RemoveAll(path string) error {
	if err := p.validate(path); err != nil {
		return err
	}
	return p.source.RemoveAll(path)
}

func (p *FilePredicateFs) Remove(path string) error {
	if err := p.validate(path); err != nil {
		return err
	}
	return p.source.Remove(path)
}

func (p *FilePredicateFs) OpenFile(path string, flag int, perm os.FileMode) (File, error) {
	if err := p.validate(path); err != nil {
		return nil, err
	}
	return p.source.OpenFile(path, flag, perm)
}

func (p *FilePredicateFs) Open(path string) (File, error) {
	if err := p.validate(path); err != nil {
		return nil, err
	}
	f, err := p.source.Open(path)
	if err != nil {
		return nil, err
	}
	return &PredicateFile{f: f, pred: p.pred}, nil
}

func (p *FilePredicateFs) Mkdir(n string, path os.FileMode) error {
	return p.source.Mkdir(n, path)
}

func (p *FilePredicateFs) MkdirAll(n string, path os.FileMode) error {
	return p.source.MkdirAll(n, path)
}

func (p *FilePredicateFs) Create(path string) (File, error) {
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

func (f *PredicateFile) Readdir(c int) (filtered []os.FileInfo, err error) {
	var infos []os.FileInfo
	infos, err = f.f.Readdir(c)
	if err != nil {
		return nil, err
	}
	for _, i := range infos {
		if f.pred(i.IsDir(), filepath.Join(f.f.Name(), i.Name())) {
			filtered = append(filtered, i)
		}
	}
	return filtered, nil
}

func (f *PredicateFile) ReadDir(n int) (filtered []fs.DirEntry, err error) {
	var entreis []fs.DirEntry
	if rdf, ok := f.f.(fs.ReadDirFile); ok {
		entreis, err = rdf.ReadDir(n)
	} else {
		entreis, err = readDirFile{f.f}.ReadDir(n)
	}
	if err != nil {
		return nil, err
	}
	for _, e := range entreis {
		if f.pred(e.IsDir(), filepath.Join(f.f.Name(), e.Name())) {
			filtered = append(filtered, e)
		}
	}
	return filtered, nil
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
