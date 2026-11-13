package afero

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// The shared tests in Fss use absolute temp paths, so the root is "/".
// On Windows these paths start with a drive letter, which os.Root rejects.
func init() {
	if runtime.GOOS == "windows" {
		return
	}
	root, err := os.OpenRoot("/")
	if err != nil {
		panic(err)
	}
	Fss = append(Fss, NewOsRootFs(root))
}

// newOsRootFs returns an OsRootFs rooted at a new directory inside a parent
// temporary directory. It returns the fs, the root directory and the parent.
func newOsRootFs(t *testing.T) (Fs, string, string) {
	t.Helper()
	parent := t.TempDir()
	dir := filepath.Join(parent, "root")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Close() })
	return NewOsRootFs(root), dir, parent
}

func TestOsRootFsReadWrite(t *testing.T) {
	fs, dir, _ := newOsRootFs(t)

	if err := fs.MkdirAll("/a/b", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(fs, "/a/b/file.txt", []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "a", "b", "file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}

	got, err = ReadFile(fs, "a/b/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}

	info, err := fs.Stat("/")
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Error("expected / to be a directory")
	}

	entries, err := ReadDir(fs, "/")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "a" {
		t.Errorf("unexpected entries in /: %v", entries)
	}

	if err := fs.Rename("a/b/file.txt", "a/renamed.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat("a/renamed.txt"); err != nil {
		t.Error(err)
	}
	if err := fs.Remove("a/renamed.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat("a/renamed.txt"); !os.IsNotExist(err) {
		t.Errorf("expected file to be removed, got %v", err)
	}
	if err := fs.RemoveAll("a"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a")); !os.IsNotExist(err) {
		t.Errorf("expected directory to be removed, got %v", err)
	}
}

func TestOsRootFsEscape(t *testing.T) {
	fs, dir, parent := newOsRootFs(t)
	checkNoEscape(t, fs, dir, parent)
}

// checkNoEscape makes sure that fs, rooted at dir inside parent, cannot
// read or write parent through "..", or through a symlink in dir.
func checkNoEscape(t *testing.T, fs Fs, dir, parent string) {
	t.Helper()
	secret := filepath.Join(parent, "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"../secret.txt", "/../secret.txt", "a/../../secret.txt"} {
		if f, err := fs.Open(name); err == nil {
			f.Close()
			t.Errorf("Open(%q) should fail", name)
		}
	}

	if f, err := fs.Create("../created.txt"); err == nil {
		f.Close()
		t.Error("Create outside the root should fail")
	}
	if _, err := os.Stat(filepath.Join(parent, "created.txt")); !os.IsNotExist(err) {
		t.Errorf("file was created outside the root: %v", err)
	}

	if err := os.Symlink(parent, filepath.Join(dir, "link")); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}
	if _, err := ReadFile(fs, "link/secret.txt"); err == nil {
		t.Error("reading through a symlink that points outside the root should fail")
	}
	if err := WriteFile(fs, "link/new.txt", []byte("x"), 0o644); err == nil {
		t.Error("writing through a symlink that points outside the root should fail")
	}
	if _, err := os.Stat(filepath.Join(parent, "new.txt")); !os.IsNotExist(err) {
		t.Errorf("file was created outside the root: %v", err)
	}
}

func TestOsRootFsSymlinker(t *testing.T) {
	fs, _, _ := newOsRootFs(t)

	s, ok := fs.(Symlinker)
	if !ok {
		t.Fatal("OsRootFs should implement Symlinker")
	}
	if err := WriteFile(fs, "target.txt", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.SymlinkIfPossible("target.txt", "/link"); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	target, err := s.ReadlinkIfPossible("link")
	if err != nil {
		t.Fatal(err)
	}
	if target != "target.txt" {
		t.Errorf("got %q, want %q", target, "target.txt")
	}

	info, called, err := s.LstatIfPossible("link")
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("expected Lstat to be called")
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected a symlink")
	}
}

func TestOsRootFsNotExist(t *testing.T) {
	fs, _, _ := newOsRootFs(t)

	_, err := fs.Open("missing.txt")
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected os.ErrNotExist, got %v", err)
	}
}

// newRootDirs returns a new directory "root" inside a parent temporary
// directory, and the parent.
func newRootDirs(t *testing.T) (string, string) {
	t.Helper()
	parent := t.TempDir()
	dir := filepath.Join(parent, "root")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir, parent
}

func TestNewRootFsOsFs(t *testing.T) {
	dir, parent := newRootDirs(t)

	fs, err := NewRootFs(NewOsFs(), dir)
	if err != nil {
		t.Fatal(err)
	}
	defer fs.Close()
	if _, ok := fs.(*OsRootFs); !ok {
		t.Fatalf("expected *OsRootFs, got %T", fs)
	}
	checkNoEscape(t, fs, dir, parent)
}

func TestNewRootFsBasePathFs(t *testing.T) {
	dir, parent := newRootDirs(t)
	base := NewBasePathFs(NewOsFs(), parent)

	fs, err := NewRootFs(base, "root")
	if err != nil {
		t.Fatal(err)
	}
	defer fs.Close()
	if _, ok := fs.(*OsRootFs); !ok {
		t.Fatalf("expected *OsRootFs, got %T", fs)
	}
	checkNoEscape(t, fs, dir, parent)

	if fs, err := NewRootFs(NewBasePathFs(NewOsFs(), dir), ".."); err == nil {
		fs.Close()
		t.Error("NewRootFs outside the base path should fail")
	}
}

func TestOsRootFsOpenRoot(t *testing.T) {
	fs, dir, _ := newOsRootFs(t)

	if err := fs.MkdirAll("sub", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(fs, "sub/a.txt", []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}

	sub, err := NewRootFs(fs, "sub")
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Close()
	if _, ok := sub.(*OsRootFs); !ok {
		t.Fatalf("expected *OsRootFs, got %T", sub)
	}
	got, err := ReadFile(sub, "a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "a" {
		t.Errorf("got %q, want %q", got, "a")
	}
	checkNoEscape(t, sub, filepath.Join(dir, "sub"), dir)
}

func TestNewRootFsFallback(t *testing.T) {
	source := NewMemMapFs()
	if err := source.MkdirAll("/sub", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(source, "/outside.txt", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	fs, err := NewRootFs(source, "/sub")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := fs.(*OsRootFs); ok {
		t.Fatal("expected the BasePathFs fallback, got *OsRootFs")
	}
	if _, ok := fs.(Symlinker); !ok {
		t.Error("the fallback should keep the Symlinker methods of BasePathFs")
	}

	if err := WriteFile(fs, "/a.txt", []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Stat("/sub/a.txt"); err != nil {
		t.Errorf("file should be inside /sub of the source: %v", err)
	}
	if _, err := ReadFile(fs, "../outside.txt"); err == nil {
		t.Error("reading outside the root should fail")
	}
	if err := fs.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestNewRootFsErrors(t *testing.T) {
	dir, _ := newRootDirs(t)
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	mem := NewMemMapFs()
	if err := WriteFile(mem, file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, source := range []Fs{NewOsFs(), mem} {
		for _, path := range []string{filepath.Join(dir, "missing"), file} {
			fs, err := NewRootFs(source, path)
			if err == nil {
				t.Errorf("%s: NewRootFs(%q) should fail", source.Name(), path)
			}
			if fs != nil {
				t.Errorf(
					"%s: NewRootFs(%q) should return a nil Fs, got %#v",
					source.Name(),
					path,
					fs,
				)
			}
		}
	}
}

func TestOsRootFsClose(t *testing.T) {
	dir, _ := newRootDirs(t)

	owned, err := NewRootFs(NewOsFs(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := owned.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := owned.Stat("/"); err == nil {
		t.Error("a root opened by NewRootFs should be closed after Close")
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	notOwned := NewOsRootFs(root)
	if err := notOwned.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := notOwned.Stat("/"); err != nil {
		t.Errorf("a root passed to NewOsRootFs should stay open after Close: %v", err)
	}
}
