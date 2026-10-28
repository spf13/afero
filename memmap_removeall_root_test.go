package afero

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMemMapFsRemoveAllRoot(t *testing.T) {
	fs := NewMemMapFs()
	root := FilePathSeparator
	directory := filepath.Join(root, "parent", "nested")
	if err := fs.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	files := []string{
		filepath.Join(root, "file"),
		filepath.Join(directory, "child"),
		filepath.Join(root, "parent-sibling"),
	}
	for _, name := range files {
		if err := WriteFile(fs, name, []byte("contents"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := fs.RemoveAll(root); err != nil {
		t.Fatal(err)
	}

	for _, name := range append(files, filepath.Join(root, "parent"), directory) {
		if _, err := fs.Stat(name); !os.IsNotExist(err) {
			t.Errorf("Stat(%q) after removing root: got %v, want not-exist", name, err)
		}
	}
	if err := fs.RemoveAll(root); err != nil {
		t.Fatalf("RemoveAll on the missing root: %v", err)
	}
}

func TestMemMapFsRemoveAllPreservesSiblingPrefix(t *testing.T) {
	fs := NewMemMapFs()
	directory := filepath.Join(FilePathSeparator, "parent")
	sibling := directory + "-sibling"
	for _, name := range []string{directory, sibling} {
		if err := fs.MkdirAll(name, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := WriteFile(
			fs,
			filepath.Join(name, "file"),
			[]byte("contents"),
			0o644,
		); err != nil {
			t.Fatal(err)
		}
	}

	if err := fs.RemoveAll(directory); err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat(filepath.Join(directory, "file")); !os.IsNotExist(err) {
		t.Errorf("removed child: got %v, want not-exist", err)
	}
	if contents, err := ReadFile(
		fs,
		filepath.Join(sibling, "file"),
	); err != nil ||
		string(contents) != "contents" {
		t.Errorf("sibling changed: contents=%q err=%v", contents, err)
	}
}
