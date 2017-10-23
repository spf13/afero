package sftpfs

import (
	"io"
	"os"
	"time"
)

// SFTPClient represents the interface to an SFTP Client.
// The github.com/pkg/sftp package client implements this interface.
type SFTPClient interface {
	Mkdir(path string) error
	Chmod(path string, mode os.FileMode) error
	Remove(path string) error
	Rename(oldname, newname string) error
	Stat(p string) (os.FileInfo, error)
	Lstat(p string) (os.FileInfo, error)
	Chtimes(path string, atime time.Time, mtime time.Time) error
	Create(path string) (SFTPFile, error)
	Open(path string) (SFTPFile, error)
}

// SFTPFile represents the interface for a file accessed via the SFTPClient.
type SFTPFile interface {
	io.Reader
	io.Writer
	io.Seeker
	io.Closer

	Stat() (os.FileInfo, error)
	Name() string
	Truncate(size int64) error
}
