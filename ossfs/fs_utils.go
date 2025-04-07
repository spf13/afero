package ossfs

import (
	"strings"
)

func (fs *Fs) isDir(s string) bool {
	return strings.HasSuffix(s, fs.separator)
}

func (fs *Fs) ensureAsDir(s string) string {
	s = fs.normFileName(s)
	if !strings.HasSuffix(s, fs.separator) {
		s = s + fs.separator
	}
	return s
}

func (fs *Fs) normFileName(s string) string {
	s = strings.TrimLeft(s, "/\\")
	s = strings.Replace(s, "\\", fs.separator, -1)
	s = strings.Replace(s, "/", fs.separator, -1)
	return s
}
