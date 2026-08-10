//go:build !windows && !(tinygo && js)

package afero

import "os"

func (OsFs) Link(oldname, newname string) error {
	return os.Link(oldname, newname)
}
