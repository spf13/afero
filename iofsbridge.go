package afero

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"
)

// errReadOnlyBridge is returned when a write operation is attempted on the read-only FS.
// It wraps os.ErrPermission.
var errReadOnlyBridge = fmt.Errorf("BridgeIOFS is read-only: %w", os.ErrPermission)

var ErrNoSeek = fmt.Errorf("underlying fs.File does not support Seek: %w", os.ErrInvalid)
var ErrNoReadAt = fmt.Errorf("underlying fs.File does not support ReadAt: %w", os.ErrInvalid)
var ErrNotDir = fmt.Errorf("not a directory: %w", os.ErrInvalid)
var ErrNoLock = fmt.Errorf("underlying fs.File does not support Lock/Unlock: %w", os.ErrInvalid)

// BridgeIOFS is a bridge that adapts an io/fs.FS to an afero.Fs.
// Since io/fs.FS is defined as read-only, this implementation is also read-only.
type BridgeIOFS struct {
	backend fs.FS
}

// NewBridgeIOFS creates a new read-only afero.Fs from an io/fs.FS.
func NewBridgeIOFS(backend fs.FS) Fs {
	return &BridgeIOFS{backend: backend}
}

// Name returns the name of the filesystem, including the underlying type for debugging.
func (f *BridgeIOFS) Name() string {
	return fmt.Sprintf("BridgeIOFS(%T)", f.backend)
}

// Stat returns the FileInfo structure describing the file.
func (f *BridgeIOFS) Stat(name string) (fs.FileInfo, error) {
	// fs.Stat efficiently handles checking if the backend implements fs.StatFS.
	return fs.Stat(f.backend, name)
}

// Open opens the named file for reading.
func (f *BridgeIOFS) Open(name string) (File, error) {
	file, err := f.backend.Open(name)
	if err != nil {
		return nil, err
	}
	// Wrap the fs.File in our adapter. Afero requires the File object to have a Name() method.
	return &bridgeFile{file: file, name: name}, nil
}

// OpenFile opens the named file. It enforces read-only access.
func (f *BridgeIOFS) OpenFile(name string, flag int, perm fs.FileMode) (File, error) {
	// Check flags to ensure no write modes are requested. O_RDONLY is 0.
	if flag&(os.O_WRONLY|os.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC) != 0 {
		return nil, errReadOnlyBridge
	}
	return f.Open(name)
}

// --- Write operations (return read-only error) ---

func (f *BridgeIOFS) Create(name string) (File, error)             { return nil, errReadOnlyBridge }
func (f *BridgeIOFS) Mkdir(name string, perm fs.FileMode) error    { return errReadOnlyBridge }
func (f *BridgeIOFS) MkdirAll(path string, perm fs.FileMode) error { return errReadOnlyBridge }
func (f *BridgeIOFS) Remove(name string) error                     { return errReadOnlyBridge }
func (f *BridgeIOFS) RemoveAll(path string) error                  { return errReadOnlyBridge }
func (f *BridgeIOFS) Rename(oldname, newname string) error         { return errReadOnlyBridge }
func (f *BridgeIOFS) Chmod(name string, mode fs.FileMode) error    { return errReadOnlyBridge }
func (f *BridgeIOFS) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return errReadOnlyBridge
}
func (f *BridgeIOFS) Chown(name string, uid, gid int) error { return errReadOnlyBridge }

// === File Implementation ===

// bridgeFile wraps fs.File to implement afero.File (read-only).
type bridgeFile struct {
	file fs.File
	name string // Store the name used to open the file.
}

func (f *bridgeFile) Close() error               { return f.file.Close() }
func (f *bridgeFile) Stat() (fs.FileInfo, error) { return f.file.Stat() }
func (f *bridgeFile) Name() string               { return f.name }
func (f *bridgeFile) Read(p []byte) (int, error) { return f.file.Read(p) }

// ReadAt attempts to use io.ReaderAt if the underlying fs.File supports it.
func (f *bridgeFile) ReadAt(p []byte, off int64) (n int, err error) {
	if r, ok := f.file.(io.ReaderAt); ok {
		return r.ReadAt(p, off)
	}
	// io/fs.File does not guarantee ReaderAt support. Use Afero standard error wrapped in PathError.
	return 0, &os.PathError{Op: "readat", Path: f.name, Err: ErrNoReadAt}
}

// Seek attempts to use io.Seeker if the underlying fs.File supports it.
func (f *bridgeFile) Seek(offset int64, whence int) (int64, error) {
	if s, ok := f.file.(io.Seeker); ok {
		return s.Seek(offset, whence)
	}
	// io/fs.File does not guarantee Seeker support. Use Afero standard error wrapped in PathError.
	return 0, &os.PathError{Op: "seek", Path: f.name, Err: ErrNoSeek}
}

// Readdir reads the contents of the directory.
func (f *bridgeFile) Readdir(count int) ([]fs.FileInfo, error) {
	// According to io/fs.File spec, if the opened file is a directory, it should implement fs.ReadDirFile.
	d, ok := f.file.(fs.ReadDirFile)
	if !ok {
		// If it doesn't implement the interface, treat it as not a directory.
		return nil, &os.PathError{Op: "readdir", Path: f.name, Err: ErrNotDir}
	}

	// ReadDir returns []fs.DirEntry
	entries, err := d.ReadDir(count)
	if err != nil {
		return nil, err
	}

	// Convert []fs.DirEntry to []fs.FileInfo (required by Afero)
	infos := make([]fs.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			// If we cannot get FileInfo from DirEntry (e.g., broken symlink).
			return nil, fmt.Errorf("error getting FileInfo for %s: %w", entry.Name(), err)
		}
		infos = append(infos, info)
	}
	return infos, nil
}

// Readdirnames reads the names of the entries in the directory.
func (f *bridgeFile) Readdirnames(count int) ([]string, error) {
	d, ok := f.file.(fs.ReadDirFile)
	if !ok {
		return nil, &os.PathError{Op: "readdirnames", Path: f.name, Err: ErrNotDir}
	}

	entries, err := d.ReadDir(count)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names, nil
}

// Sync is generally a no-op for a read-only file, but we pass it through if the underlying file supports it.
func (f *bridgeFile) Sync() error {
	if s, ok := f.file.(interface{ Sync() error }); ok {
		return s.Sync()
	}
	return nil
}

// Lock and Unlock are not supported by io/fs.
func (f *bridgeFile) Lock() error   { return ErrNoLock }
func (f *bridgeFile) Unlock() error { return ErrNoLock }

// --- Write operations for the file (disabled) ---

func (f *bridgeFile) Write(p []byte) (int, error)              { return 0, errReadOnlyBridge }
func (f *bridgeFile) WriteAt(p []byte, off int64) (int, error) { return 0, errReadOnlyBridge }
func (f *bridgeFile) Truncate(size int64) error                { return errReadOnlyBridge }
func (f *bridgeFile) WriteString(s string) (int, error)        { return 0, errReadOnlyBridge }
func (f *bridgeFile) Chmod(mode os.FileMode) error             { return errReadOnlyBridge }
func (f *bridgeFile) Chown(uid, gid int) error                 { return errReadOnlyBridge }
