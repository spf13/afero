//go:build tinygo && js

package afero

import "syscall"

// TinyGo's syscall package does not define EBADFD on js/wasm; EBADF is the
// closest available errno.
const BADFD = syscall.EBADF
