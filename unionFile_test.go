package afero

import (
	"io"
	"testing"
)

func TestUnionFileReaddir(t *testing.T) {
	base := &MemMapFs{}
	layer := &MemMapFs{}

	WriteFile(base, "foo", []byte{'F'}, 0)
	WriteFile(layer, "bar", []byte{'B'}, 0)

	openUnionFile := func() (*UnionFile, error) {
		basef, err := base.Open("/")
		if err != nil {
			return nil, err
		}
		layerf, err := layer.Open("/")
		if err != nil {
			basef.Close()
			return nil, err
		}
		return &UnionFile{
			Base:  basef,
			Layer: layerf,
		}, nil
	}

	t.Run("Readdir(-1)", func(t *testing.T) {
		f, err := openUnionFile()
		if err != nil {
			t.Error(err)
			return
		}
		defer f.Close()
		files, err := f.Readdir(-1)
		if err != nil {
			t.Error(err)
			return
		}
		if len(files) != 2 {
			t.Fatal("invalid files returned from Readdir")
			return
		}
		_, err = f.Readdir(1)
		if err != io.EOF {
			t.Fatal("Readdir did not return EOF")
			return
		}
	})

	t.Run("Readdir(1000)", func(t *testing.T) {
		f, err := openUnionFile()
		if err != nil {
			t.Error(err)
			return
		}
		defer f.Close()
		files, err := f.Readdir(1000)
		if err != nil {
			t.Error(err)
			return
		}
		if len(files) != 2 {
			t.Fatal("invalid files returned Readdir")
			return
		}
		_, err = f.Readdir(1)
		if err != io.EOF {
			t.Fatal("Readdir did not return EOF")
			return
		}
	})

	t.Run("Readdir(1) x 2", func(t *testing.T) {
		f, err := openUnionFile()
		if err != nil {
			t.Error(err)
			return
		}
		defer f.Close()
		files1, err := f.Readdir(1)
		if err != nil {
			t.Error(err)
			return
		}
		if len(files1) != 1 {
			t.Fatal("invalid files returned from Readdir")
			return
		}
		files2, err := f.Readdir(1)
		if err != nil {
			t.Error(err)
			return
		}
		if len(files2) != 1 {
			t.Fatal("invalid files returned from Readdir")
			return
		}
		_, err = f.Readdir(1)
		if err != io.EOF {
			t.Fatal("Readdir did not return EOF")
			return
		}
	})

}
