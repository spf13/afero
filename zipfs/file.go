package zipfs

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/spf13/afero"
)

type File struct {
	fs            *Fs
	zipfile       *zip.File
	pseudodir     *pseudoDir
	reader        io.ReadCloser
	offset        int64
	isdir, closed bool
	buf           []byte
}

func (f *File) fillBuffer(offset int64) (err error) {
	if offset < 0 {
		return syscall.EINVAL
	}
	if f.reader == nil {
		if f.reader, err = f.zipfile.Open(); err != nil {
			return
		}
	}
	if offset > int64(f.zipfile.UncompressedSize64) {
		offset = int64(f.zipfile.UncompressedSize64)
		err = io.EOF
	}
	if len(f.buf) >= int(offset) {
		return
	}
	buf := make([]byte, int(offset)-len(f.buf))
	n, readErr := io.ReadFull(f.reader, buf)
	if n > 0 {
		f.buf = append(f.buf, buf[:n]...)
	}
	if readErr != nil {
		if readErr == io.ErrUnexpectedEOF {
			err = io.EOF
		} else {
			err = readErr
		}
	}
	return
}

func (f *File) Close() (err error) {
	f.zipfile = nil
	f.closed = true
	f.buf = nil
	if f.reader != nil {
		err = f.reader.Close()
		f.reader = nil
	}
	return
}

func (f *File) Read(p []byte) (n int, err error) {
	if f.isdir {
		return 0, syscall.EISDIR
	}
	if f.closed {
		return 0, afero.ErrFileClosed
	}
	if len(p) == 0 {
		return 0, nil
	}
	fillErr := f.fillBuffer(f.offset + int64(len(p)))
	if f.offset >= int64(len(f.buf)) {
		if fillErr != nil {
			return 0, fillErr
		}
		return 0, io.EOF
	}
	n = copy(p, f.buf[f.offset:])
	f.offset += int64(n)
	if n < len(p) {
		if fillErr != nil {
			err = fillErr
		} else {
			err = io.EOF
		}
	}
	return
}

func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	if f.isdir {
		return 0, syscall.EISDIR
	}
	if f.closed {
		return 0, afero.ErrFileClosed
	}
	if off < 0 {
		return 0, syscall.EINVAL
	}
	if len(p) == 0 {
		return 0, nil
	}
	fillErr := f.fillBuffer(off + int64(len(p)))
	if off >= int64(len(f.buf)) {
		if fillErr != nil {
			return 0, fillErr
		}
		return 0, io.EOF
	}
	n = copy(p, f.buf[int(off):])
	if n < len(p) {
		if fillErr != nil {
			err = fillErr
		} else {
			err = io.EOF
		}
	}
	return
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	if f.isdir {
		return 0, syscall.EISDIR
	}
	if f.closed {
		return 0, afero.ErrFileClosed
	}
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		offset += f.offset
	case io.SeekEnd:
		offset += int64(f.zipfile.UncompressedSize64)
	default:
		return 0, syscall.EINVAL
	}
	if offset < 0 || offset > int64(f.zipfile.UncompressedSize64) {
		return 0, afero.ErrOutOfRange
	}
	f.offset = offset
	return offset, nil
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
