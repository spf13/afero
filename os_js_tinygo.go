//go:build tinygo && js

package afero

import (
	"os"
	"syscall"
)

// TinyGo's os package does not define Chmod, Chown or Link on the js/wasm
// target, so afero does not compile there at all. There is no filesystem to
// act on either, so these report ENOSYS rather than pretending to succeed.
//
// Only js/wasm is affected: TinyGo's linux and wasip1 targets provide all
// three, and continue to use the standard implementations.

func osChmod(name string, mode os.FileMode) error { return syscall.ENOSYS }
func osChown(name string, uid, gid int) error     { return syscall.ENOSYS }

func (OsFs) Link(oldname, newname string) error { return syscall.ENOSYS }
