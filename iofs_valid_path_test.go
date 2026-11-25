package afero

import (
	"errors"
	"io/fs"
	"testing"
)

func TestIOFSSubValidPath(t *testing.T) {
	base := NewMemMapFs()
	if err := base.MkdirAll("public", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(base, "public/file.txt", []byte("public"), 0o644); err != nil {
		t.Fatal(err)
	}
	iofs := NewIOFS(base)
	for _, dir := range []string{"", "/public", "public/", "public/..", "../public", "public//nested"} {
		t.Run(dir, func(t *testing.T) {
			sub, err := iofs.Sub(dir)
			if sub != nil || !errors.Is(err, fs.ErrInvalid) {
				t.Fatalf("Sub(%q) = %v, %v; want nil, fs.ErrInvalid", dir, sub, err)
			}
		})
	}
	for _, dir := range []string{".", "public"} {
		t.Run("valid/"+dir, func(t *testing.T) {
			sub, err := iofs.Sub(dir)
			if err != nil {
				t.Fatal(err)
			}
			name := "file.txt"
			if dir == "." {
				name = "public/file.txt"
			}
			content, err := fs.ReadFile(sub, name)
			if err != nil || string(content) != "public" {
				t.Fatalf("ReadFile = %q, %v", content, err)
			}
		})
	}
}

func TestIOFSStatValidPath(t *testing.T) {
	base := NewMemMapFs()
	if err := base.MkdirAll("public", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(base, "public/file.txt", []byte("public"), 0o644); err != nil {
		t.Fatal(err)
	}
	iofs := NewIOFS(base)
	for _, name := range []string{"", "/public", "public/", "public/../public/file.txt", "public//file.txt"} {
		t.Run(name, func(t *testing.T) {
			info, err := fs.Stat(iofs, name)
			if info != nil || !errors.Is(err, fs.ErrInvalid) {
				t.Fatalf("Stat(%q) = %v, %v; want nil, fs.ErrInvalid", name, info, err)
			}
		})
	}
	info, err := fs.Stat(iofs, "public/file.txt")
	if err != nil || info.Name() != "file.txt" {
		t.Fatalf("valid Stat = %v, %v", info, err)
	}
}
