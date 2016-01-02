package afero

import (
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestUnionCreateExisting(t *testing.T) {
	base := &MemMapFs{}
	roBase := NewFilter(base)
	roBase.AddFilter(NewReadonlyFilter())

	ufs := NewUnionFs(roBase, &MemMapFs{}, NewCoWUnionFs())

	base.MkdirAll("/home/test", 0777)
	fh, _ := base.Create("/home/test/file.txt")
	fh.WriteString("This is a test")
	fh.Close()

	fh, err := ufs.OpenFile("/home/test/file.txt", os.O_RDWR, 0666)
	if err != nil {
		t.Errorf("Failed to open file r/w: %s", err)
	}

	_, err = fh.Write([]byte("####"))
	if err != nil {
		t.Errorf("Failed to write file: %s", err)
	}
	fh.Seek(0, 0)
	data, err := ioutil.ReadAll(fh)
	if err != nil {
		t.Errorf("Failed to read file: %s", err)
	}
	if string(data) != "#### is a test" {
		t.Errorf("Got wrong data")
	}
	fh.Close()

	fh, _ = base.Open("/home/test/file.txt")
	data, err = ioutil.ReadAll(fh)
	if string(data) != "This is a test" {
		t.Errorf("Got wrong data in base file")
	}
	fh.Close()

	fh, err = ufs.Create("/home/test/file.txt")
	switch err {
	case nil:
		if fi, _ := fh.Stat(); fi.Size() != 0 {
			t.Errorf("Create did not truncate file")
		}
		fh.Close()
	default:
		t.Errorf("Create failed on existing file")
	}

}

func TestUnionMergeReaddir(t *testing.T) {
	base := &MemMapFs{}
	roBase := NewFilter(base)
	roBase.AddFilter(NewReadonlyFilter())

	ufs := NewUnionFs(roBase, &MemMapFs{}, NewCoWUnionFs())

	base.MkdirAll("/home/test", 0777)
	fh, _ := base.Create("/home/test/file.txt")
	fh.WriteString("This is a test")
	fh.Close()

	fh, _ = ufs.Create("/home/test/file2.txt")
	fh.WriteString("This is a test")
	fh.Close()

	fh, _ = ufs.Open("/home/test")
	files, err := fh.Readdirnames(-1)
	if err != nil {
		t.Errorf("Readdirnames failed")
	}
	if len(files) != 2 {
		t.Errorf("Got wrong number of files: %v", files)
	}
}

func TestUnionCacheWrite(t *testing.T) {
	base := &MemMapFs{}
	layer := &MemMapFs{}
	ufs := NewUnionFs(base, layer, NewCacheUnionFs(0))

	base.Mkdir("/data", 0777)

	fh, err := ufs.Create("/data/file.txt")
	if err != nil {
		t.Errorf("Failed to create file")
	}
	_, err = fh.Write([]byte("This is a test"))
	if err != nil {
		t.Errorf("Failed to write file")
	}

	fh.Seek(0, os.SEEK_SET)
	buf := make([]byte, 4)
	_, err = fh.Read(buf)
	fh.Write([]byte(" IS A"))
	fh.Close()

	baseData, _ := ReadFile(base, "/data/file.txt")
	layerData, _ := ReadFile(layer, "/data/file.txt")
	if string(baseData) != string(layerData) {
		t.Errorf("Different data: %s <=> %s", baseData, layerData)
	}
}

func TestUnionCacheExpire(t *testing.T) {
	base := &MemMapFs{}
	layer := &MemMapFs{}
	ufs := NewUnionFs(base, layer, NewCacheUnionFs(1*time.Second))

	base.Mkdir("/data", 0777)

	fh, err := ufs.Create("/data/file.txt")
	if err != nil {
		t.Errorf("Failed to create file")
	}
	_, err = fh.Write([]byte("This is a test"))
	if err != nil {
		t.Errorf("Failed to write file")
	}
	fh.Close()

	fh, _ = base.Create("/data/file.txt")
	fh.WriteString("Another test")
	fh.Close()

	time.Sleep(1 * time.Second)
	data, _ := ReadFile(ufs, "/data/file.txt")
	if string(data) != "Another test" {
		t.Errorf("cache time failed")
	}
}
