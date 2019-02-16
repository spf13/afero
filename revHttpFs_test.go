package afero

import "testing"

func TestRevHttpFs(t *testing.T) {
	aferoFs := NewMemMapFs()
	aferoFs.Create("myfile.txt")
	aferoFs.Create("myfile2.txt")

	newfs := NewReverseHttpFs(NewHttpFs(aferoFs))

	// This ensures that ReverseHttpFs matches the Fs interface
	NewReadOnlyFs(newfs)

	f, err := newfs.Open("/")
	if err != nil {
		t.Errorf("Failed to open root directory")
	}
	n, err := f.Readdirnames(-1)
	if err != nil {
		t.Errorf("Failed to read dir names")
	}
	if len(n) != 2 {
		t.Errorf("Filesystem does not read directories correctly")
	}
	if n[0] != "myfile.txt" && n[0] != "myfile2.txt" {
		t.Errorf("File %s not matching", n[0])
	}
	if n[1] != "myfile.txt" && n[1] != "myfile2.txt" {
		t.Errorf("File %s not matching", n[1])
	}
	if n[0] == n[1] {
		t.Errorf("Both files have same name: %s", n[0])
	}

}
