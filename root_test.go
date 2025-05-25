package afero

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// newFs creates a *MemMapFs with some files for testing
func newFs(t testing.TB) Fs {
	t.Helper()
	fs := NewMemMapFs()
	createAndFill := func(path, name string) error {
		t.Helper()
		f2, err := os.Open(name)
		if err != nil {
			return err
		}
		f1, err := fs.Create(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(f1, f2)
		return err
	}
	fs.MkdirAll("/home/burt", os.ModeDir)
	fs.MkdirAll("/home/ernie/Documents", os.ModeDir)
	fs.MkdirAll("/home/ernie/Pictures/dogs", os.ModeDir)
	err := createAndFill("/home/ernie/Pictures/dogs/fido.jpg", "testdata/fido.jpg")
	if err != nil {
		t.Error(err)
	}
	err = createAndFill("/home/ernie/Pictures/dogs/rex.jpg", "testdata/rex.jpg")
	if err != nil {
		t.Error(err)
	}
	err = createAndFill("/home/ernie/Documents", "testdata/sales.xls")
	if err != nil {
		t.Error(err)
	}
	return fs
}

// mustBytes reads out bytes from files in testdata/
func mustBytes(t testing.TB, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Errorf("mustBytes failed. %s", err)
	}
	return b
}

func TestRoot(t *testing.T) {
	t.Parallel()

	parent := newFs(t)

	//	sanity check
	info, err := parent.Stat("/home/burt")
	if err != nil {
		t.Errorf("burt/Documents should exist. %s", err)
	}
	if !info.IsDir() {
		t.Error("burt/Documents should be a directory")
	}

	ernie, err := parent.OpenRoot("/home/ernie")
	if err != nil {
		t.Errorf("it should be possible to create this root. %s", err)
	}
	assert.Equal(t, "/home/ernie", ernie.Name())

	//	try to escape. see that it fails
	noBurt, err := ernie.Open("../burt")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidRoot, fmt.Sprintf("expected ErrInvalidRoot but got: %s", err))
	assert.Nil(t, noBurt)

	noFortune, err := ernie.FS().Stat("Documents/fortune.txt")
	assert.Error(t, err)
	assert.Nil(t, noFortune, "this file should not exist")

	//	write a file from parent
	WriteFile(parent, "/home/ernie/Documents/fortune.txt", mustBytes(t, "fortune.txt"), os.ModeType)

	//	read it from a child
	yesFortune, err := ernie.FS().Stat("Documents/fortune.txt")
	assert.NoError(t, err)
	assert.NotNil(t, yesFortune, "this file should exist")
	assert.Equal(t, "fortune.txt", yesFortune.Name())

	//	delete a file in the child
	err = ernie.FS().Remove("Pictures/dogs/rex.jpg")
	assert.NoError(t, err)

	//	see that it is gone in the parent
	info, err = parent.Stat("/home/ernie/Pictures/dogs/rex.jpg")
	assert.Error(t, err, "rex should be gone")
	assert.Nil(t, info)

	//	create a child from a child
	dogs, err := ernie.OpenRoot("Pictures/dogs")
	assert.NoError(t, err)
	dirEntries, err := ReadDir(dogs.FS(), ".")
	assert.NoError(t, err)
	assert.Len(t, dirEntries, 1)
	assert.Equal(t, "fido.jpg", dirEntries[0].Name())

	//	the grandchild should not be able to access the child
	f, err := dogs.Open("../Documents/sales.xls")
	assert.Nil(t, f)
	assert.ErrorIs(t, err, ErrInvalidRoot)

	//	close child, and see that subsequent operations fail
	err = dogs.Close()
	assert.NoError(t, err)

	assert.Equal(t, "", dogs.Name())
	f, err = dogs.Open("fido.jpg")
	assert.Nil(t, f)
	assert.Error(t, err)
}
