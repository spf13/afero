// +build go1.16

package afero

import (
	"os"
	"testing"
	"testing/fstest"
)

func TestIOFS(t *testing.T) {
	t.Parallel()

	t.Run("use MemMapFs", func(t *testing.T) {
		mmfs := NewMemMapFs()

		err := mmfs.MkdirAll("dir1/dir2", os.ModePerm)
		if err != nil {
			t.Fatal("MkdirAll failed:", err)
		}

		f, err := mmfs.OpenFile("dir1/dir2/test.txt", os.O_RDWR|os.O_CREATE, os.ModePerm)
		if err != nil {
			t.Fatal("OpenFile (O_CREATE) failed:", err)
		}

		f.Close()

		if err := fstest.TestFS(NewIOFS(mmfs), "dir1/dir2/test.txt"); err != nil {
			t.Error(err)
		}
	})

	t.Run("use OsFs", func(t *testing.T) {
		osfs := NewBasePathFs(NewOsFs(), t.TempDir())

		err := osfs.MkdirAll("dir1/dir2", os.ModePerm)
		if err != nil {
			t.Fatal("MkdirAll failed:", err)
		}

		f, err := osfs.OpenFile("dir1/dir2/test.txt", os.O_RDWR|os.O_CREATE, os.ModePerm)
		if err != nil {
			t.Fatal("OpenFile (O_CREATE) failed:", err)
		}

		f.Close()

		if err := fstest.TestFS(NewIOFS(osfs), "dir1/dir2/test.txt"); err != nil {
			t.Error(err)
		}
	})
}
