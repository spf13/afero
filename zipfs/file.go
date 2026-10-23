package zipfs

import (
	"archive/zip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/spf13/afero"
)

type File struct {
	mu            sync.RWMutex
	fs            *Fs
	zipfile       *zip.File
	pseudodir     *pseudoDir
	stream        fileStream
	offset        int64
	isdir, closed bool
}

// fileStream keeps only the decompressor state, not the decompressed prefix.
type fileStream struct {
	reader io.ReadCloser
	pos    int64
	err    error
}

func (s *fileStream) close() error {
	if s.reader == nil {
		return nil
	}
	err := s.reader.Close()
	*s = fileStream{}
	return err
}

func (s *fileStream) read(p []byte) (int, error) {
	total := 0
	for total < len(p) {
		if s.err != nil {
			return total, s.err
		}
		n, err := s.reader.Read(p[total:])
		total += n
		s.pos += int64(n)
		s.err = err
		if err != nil {
			if err == io.EOF && total == len(p) {
				return total, nil
			}
			return total, err
		}
	}
	return total, nil
}

func (s *fileStream) position(file *zip.File, offset int64) error {
	if offset < s.pos {
		if err := s.close(); err != nil {
			return err
		}
	}
	if s.reader == nil {
		r, err := file.Open()
		if err != nil {
			return err
		}
		s.reader = r
	}
	if s.pos == offset {
		return nil
	}
	var scratch [32 * 1024]byte
	for s.pos < offset {
		n := min(int64(len(scratch)), offset-s.pos)
		if _, err := s.read(scratch[:n]); err != nil {
			return err
		}
	}
	return nil
}

func (f *File) Close() (err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.zipfile = nil
	f.closed = true
	return f.stream.close()
}

func (f *File) Read(p []byte) (n int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.isdir {
		return 0, syscall.EISDIR
	}
	if f.closed {
		return 0, afero.ErrFileClosed
	}
	if len(p) == 0 {
		return 0, nil
	}
	if err = f.stream.position(f.zipfile, f.offset); err != nil {
		return 0, err
	}
	n, err = f.stream.read(p)
	f.offset += int64(n)
	return
}

func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.isdir {
		return 0, syscall.EISDIR
	}
	if f.closed {
		return 0, afero.ErrFileClosed
	}
	if off < 0 {
		return 0, afero.ErrOutOfRange
	}
	if len(p) == 0 {
		return 0, nil
	}
	if uint64(off) >= f.zipfile.UncompressedSize64 {
		return 0, io.EOF
	}
	var stream fileStream
	defer func() {
		if closeErr := stream.close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()
	if err = stream.position(f.zipfile, off); err != nil {
		return 0, err
	}
	return stream.read(p)
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.isdir {
		return 0, syscall.EISDIR
	}
	if f.closed {
		return 0, afero.ErrFileClosed
	}
	var base int64
	size := int64(f.zipfile.UncompressedSize64)
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		base = f.offset
	case io.SeekEnd:
		base = size
	default:
		return 0, syscall.EINVAL
	}
	if offset < -base || offset > size-base {
		return 0, afero.ErrOutOfRange
	}
	f.offset = base + offset
	return f.offset, nil
}

func (f *File) Write(p []byte) (n int, err error) { return 0, syscall.EPERM }

func (f *File) WriteAt(p []byte, off int64) (n int, err error) { return 0, syscall.EPERM }

func (f *File) Name() string {
	if f.zipfile == nil {
		if f.pseudodir != nil {
			return f.pseudodir.path
		}
		return string(filepath.Separator)
	}
	return filepath.Join(splitpath(f.zipfile.Name))
}

func (f *File) getDirEntries() (map[string]*zip.File, error) {
	if !f.isdir {
		return nil, syscall.ENOTDIR
	}
	name := f.Name()
	entries, ok := f.fs.files[name]
	if !ok {
		return nil, &os.PathError{Op: "readdir", Path: name, Err: syscall.ENOENT}
	}
	return entries, nil
}

func (f *File) Readdir(count int) (fi []os.FileInfo, err error) {
	zipfiles, err := f.getDirEntries()
	if err != nil {
		return nil, err
	}
	for d, zipfile := range zipfiles {
		if zipfile == nil {
			fi = append(fi, pseudoDir{d})
		} else {
			fi = append(fi, zipfile.FileInfo())
		}
		if count > 0 && len(fi) >= count {
			break
		}
	}
	return
}

func (f *File) Readdirnames(count int) (names []string, err error) {
	zipfiles, err := f.getDirEntries()
	if err != nil {
		return nil, err
	}
	for filename := range zipfiles {
		names = append(names, filename)
		if count > 0 && len(names) >= count {
			break
		}
	}
	return
}

func (f *File) Stat() (os.FileInfo, error) {
	if f.zipfile == nil {
		if f.pseudodir != nil {
			return f.pseudodir, nil
		}
		return &pseudoRoot{}, nil
	}
	return f.zipfile.FileInfo(), nil
}

func (f *File) Sync() error { return nil }

func (f *File) Truncate(size int64) error { return syscall.EPERM }

func (f *File) WriteString(s string) (ret int, err error) { return 0, syscall.EPERM }
