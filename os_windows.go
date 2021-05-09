// +build windows

package afero

import (
	"path/filepath"
	"strings"
)

// windows cannot handle long relative paths, so convert to absolute paths by default
func normalizeLongPath(path string) string {
	// get absolute path - len(path) is not reliable for early return
	absolutePath, err := filepath.Abs(path)

	// fallback to default behaviour on error
	if err != nil {
		return path
	}

	// path is already normalized
	if strings.HasPrefix(absolutePath, `\\?\`) {
		return absolutePath
	}

	// normalize "network path"
	if strings.HasPrefix(absolutePath, `\\`) {
		return `\\?\UNC\` + strings.TrimPrefix(absolutePath, `\\`)
	}

	// normalize "local path"
	return `\\?\` + absolutePath
}
