package gcs

import (
	"errors"
	"os"
)

var (
	ErrFileClosed   = errors.New("File is closed")
	ErrOutOfRange   = errors.New("Out of range")
	ErrTooLarge     = errors.New("Too large")
	ErrFileNotFound = os.ErrNotExist
)
