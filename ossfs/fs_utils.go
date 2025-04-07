package ossfs

import (
	"io"
	"strings"
)

func (fs *Fs) isDir(s string) bool {
	return strings.HasSuffix(s, fs.separator)
}

func (fs *Fs) ensureAsDir(s string) string {
	return fs.ensureTrailingSeparator(fs.normSeparators(s))
}

func (fs *Fs) normSeparators(s string) string {
	return strings.Replace(strings.Replace(s, "\\", fs.separator, -1), "/", fs.separator, -1)
}

func (fs *Fs) ensureTrailingSeparator(s string) string {
	if len(s) > 0 && !fs.isDir(s) {
		return s + fs.separator
	}
	return s
}

func (fs *Fs) putObjectStr(name, str string) (bool, error) {
	return fs.putObjectReader(name, strings.NewReader(str))
}

func (fs *Fs) putObjectReader(name string, reader io.Reader) (bool, error) {
	return fs.manager.PutObject(fs.ctx, fs.bucketName, name, reader)
}
