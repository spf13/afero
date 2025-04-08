package ossfs

import (
	"strings"
)

func (fs *Fs) isDir(s string) bool {
	sep := fs.separator
	if fs.separator == "" {
		sep = "/"
	}
	return strings.HasSuffix(s, sep)
}

func (fs *Fs) ensureAsDir(s string) string {
	sep := fs.separator
	if fs.separator == "" {
		sep = "/"
	}
	s = fs.normFileName(s)
	if !strings.HasSuffix(s, sep) {
		s = s + sep
	}
	return s
}

func (fs *Fs) normFileName(s string) string {
	sep := fs.separator
	if fs.separator == "" {
		sep = "/"
	}
	s = strings.TrimLeft(s, "/\\")
	s = strings.Replace(s, "\\", sep, -1)
	s = strings.Replace(s, "/", sep, -1)
	return s
}
