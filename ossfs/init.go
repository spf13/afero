package ossfs

import (
	"os"

	"github.com/spf13/afero"
)

func init() {
	// Ensure oss.Fs implements afero.Fs interface
	var _ afero.Fs = (*Fs)(nil)
	var _ afero.File = (*File)(nil)
	var _ os.FileInfo = (*FileInfo)(nil)
}
