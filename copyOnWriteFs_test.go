package afero

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestCopyOnWrite(t *testing.T) {
	var fs Fs
	var err error
	fs = newCopyWriteFs()
	err = fs.MkdirAll("nonexistent/directory/", 0744)
	if err != nil {
		t.Error(err)
		return
	}
	_, err = fs.Create("nonexistent/directory/newfile")
	if err != nil {
		t.Error(err)
		return
	}
}

func newCopyWriteFs() Fs {
	base := NewOsFs()
	roBase := NewReadOnlyFs(base)
	ufs := NewCopyOnWriteFs(roBase, NewMemMapFs())
	return ufs
}

func TestStat(t *testing.T) {
	var fs Fs
	fs = newCopyWriteFs()
	// create os file
	var err error
	err = os.MkdirAll("existent/", 0744)
	if err != nil {
		t.Error(err)
		return
	}
	if err = ioutil.WriteFile("existent/file", []byte{}, 0644); err != nil {
		t.Error(err)
		return
	}
	if _, err = fs.Stat("existent/file"); err != nil {
		t.Error(err)
		return
	}
}
