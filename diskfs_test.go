package afero

import (
	"bytes"
	"io"
	"path"
	"path/filepath"
	"testing"

	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
)

func tmpDiskFs(dir string) (filesystem.FileSystem, error) {
	d, err := diskfs.Create(path.Join(dir, "test.img"), 2e6, diskfs.Raw, diskfs.SectorSizeDefault)
	if err != nil {
		return nil, err
	}

	return d.CreateFilesystem(disk.FilesystemSpec{
		Partition: 0,
		FSType:    filesystem.TypeFat32,
	})
}

func TestDiskFs(t *testing.T) {
	osFs := NewOsFs()
	writeDir, err := TempDir(osFs, "", "copy-on-write-test")
	if err != nil {
		t.Fatal("error creating tempDir", err)
	}
	defer osFs.RemoveAll(writeDir)

	fs, err := tmpDiskFs(writeDir)
	if err != nil {
		t.Fatal("error creating tempDiskFs", err)
	}

	diskFs := NewDiskFs(fs)

	// diskFs requires absolute paths
	dir := "/some/path"

	err = diskFs.MkdirAll(dir, 0o744)
	if err != nil {
		t.Fatal(err)
	}

	f, err := diskFs.Create(filepath.Join(dir, "newfile"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.WriteString("foo"); err != nil {
		t.Fatal(err)
	}

	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	f, err = diskFs.Open(filepath.Join(dir, "newfile"))
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, f); err != nil {
		t.Fatal(err)
	}

	actual := buf.String()
	if actual != "foo" {
		t.Fatalf("expected file contents to be \"foo\", got %q", actual)
	}
}
