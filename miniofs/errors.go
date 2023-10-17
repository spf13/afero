package minio

import (
	"errors"
	"syscall"
)

var (
	ErrNoBucketInName     = errors.New("no bucket name found in the name")
	ErrFileClosed         = errors.New("file is closed")
	ErrOutOfRange         = errors.New("out of range")
	ErrObjectDoesNotExist = errors.New("storage: object doesn't exist")
	ErrEmptyObjectName    = errors.New("storage: object name is empty")
	ErrFileNotFound       = syscall.ENOENT
)
