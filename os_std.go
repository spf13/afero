//go:build !(tinygo && js)

package afero

import "os"

func osChmod(name string, mode os.FileMode) error { return os.Chmod(name, mode) }
func osChown(name string, uid, gid int) error     { return os.Chown(name, uid, gid) }
