package afero_test

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/spf13/afero"
)

// Setup the mock io/fs.FS for testing using Go's standard testing FS.
func setupTestBridgeFS() fs.FS {
	return fstest.MapFS{
		"file.txt":       {Data: []byte("hello world"), Mode: 0444},
		"dir/nested.txt": {Data: []byte("nested content"), Mode: 0444},
		"empty_dir":      {Mode: fs.ModeDir | 0555},
	}
}

func TestBridgeIOFS_Read(t *testing.T) {
	afs := afero.NewBridgeIOFS(setupTestBridgeFS())

	t.Run("Name", func(t *testing.T) {
		if !strings.Contains(afs.Name(), "fstest.MapFS") {
			t.Errorf("Expected Name() to contain 'fstest.MapFS', got %s", afs.Name())
		}
	})

	t.Run("ReadFile", func(t *testing.T) {
		content, err := afero.ReadFile(afs, "file.txt")
		if err != nil {
			t.Fatalf("ReadFile failed: %v", err)
		}
		if string(content) != "hello world" {
			t.Errorf("Expected 'hello world', got '%s'", string(content))
		}
	})

	t.Run("Open and FileName", func(t *testing.T) {
		file, err := afs.Open("dir/nested.txt")
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer file.Close()
		if file.Name() != "dir/nested.txt" {
			t.Errorf("Expected file name 'dir/nested.txt', got '%s'", file.Name())
		}
	})

	t.Run("Stat", func(t *testing.T) {
		info, err := afs.Stat("file.txt")
		if err != nil {
			t.Fatalf("Stat file.txt failed: %v", err)
		}
		if info.Size() != int64(len("hello world")) {
			t.Errorf("Expected size %d, got %d", len("hello world"), info.Size())
		}
		if info.IsDir() {
			t.Error("Expected file.txt not to be a directory")
		}

		info, err = afs.Stat("dir")
		if err != nil {
			t.Fatalf("Stat dir failed: %v", err)
		}
		if !info.IsDir() {
			t.Error("Expected dir to be a directory")
		}
	})

	t.Run("FileNotFound", func(t *testing.T) {
		_, err := afs.Open("nonexistent.txt")
		if !os.IsNotExist(err) {
			t.Errorf("Expected ErrNotExist, got %v", err)
		}
	})
}

func TestBridgeIOFS_ReadDir(t *testing.T) {
	afs := afero.NewBridgeIOFS(setupTestBridgeFS())

	t.Run("Root directory", func(t *testing.T) {
		entries, err := afero.ReadDir(afs, ".")
		if err != nil {
			t.Fatalf("ReadDir failed: %v", err)
		}

		// Expecting "file.txt", "dir", "empty_dir" (MapFS returns sorted results)
		if len(entries) != 3 {
			t.Fatalf("Expected 3 entries, got %d", len(entries))
		}
		if entries[0].Name() != "dir" {
			t.Errorf("Expected entry 0 to be 'dir', got %s", entries[0].Name())
		}
		if entries[1].Name() != "empty_dir" {
			t.Errorf("Expected entry 1 to be 'empty_dir', got %s", entries[1].Name())
		}
		if entries[2].Name() != "file.txt" {
			t.Errorf("Expected entry 2 to be 'file.txt', got %s", entries[2].Name())
		}
	})

	t.Run("ReadDir on file", func(t *testing.T) {
		f, err := afs.Open("file.txt")
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer f.Close()

		// Attempting Readdir on a file should return ErrNotDir
		_, err = f.Readdir(0)
		if !errors.Is(err, afero.ErrNotDir) {
			t.Errorf("Expected afero.ErrNotDir, got %v", err)
		}
	})
}

func TestBridgeIOFS_WriteOperations(t *testing.T) {
	afs := afero.NewBridgeIOFS(setupTestBridgeFS())

	// Helper to check if the error wraps os.ErrPermission (which the internal errReadOnlyBridge does)
	checkReadOnlyError := func(t *testing.T, err error) {
		t.Helper()
		if err == nil {
			t.Error("Expected an error, got nil")
			return
		}
		// We check if it wraps os.ErrPermission, as the specific internal error isn't exposed.
		if !os.IsPermission(err) {
			t.Errorf("Expected error to wrap os.ErrPermission (indicating read-only), got %v", err)
		}
	}

	t.Run("Filesystem modifications", func(t *testing.T) {
		checkReadOnlyError(t, afs.Mkdir("newdir", 0755))
		checkReadOnlyError(t, afs.Remove("file.txt"))
		checkReadOnlyError(t, afs.Rename("file.txt", "new.txt"))
		checkReadOnlyError(t, afs.Chown("file.txt", 0, 0))
	})

	t.Run("OpenFile with write flags", func(t *testing.T) {
		// O_RDONLY should succeed
		f, err := afs.OpenFile("file.txt", os.O_RDONLY, 0)
		if err != nil {
			t.Fatalf("OpenFile O_RDONLY failed: %v", err)
		}
		f.Close()

		// Write flags should fail
		_, err = afs.OpenFile("file.txt", os.O_RDWR, 0644)
		checkReadOnlyError(t, err)
		_, err = afs.OpenFile("newfile.txt", os.O_CREATE|os.O_RDWR, 0644)
		checkReadOnlyError(t, err)
	})

	t.Run("Write to opened file", func(t *testing.T) {
		f, err := afs.Open("file.txt")
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer f.Close()

		_, err = f.Write([]byte("test"))
		checkReadOnlyError(t, err)
		err = f.Truncate(0)
		checkReadOnlyError(t, err)
	})
}

// TestComposition verifies that the BridgeIOFS can be used as a base layer in a Union FS.
func TestBridgeIOFS_Composition(t *testing.T) {
	// 1. Base layer: BridgeIOFS (read-only)
	baseFs := afero.NewBridgeIOFS(setupTestBridgeFS())

	// 2. Overlay layer: Writable memory FS
	overlayFs := afero.NewMemMapFs()
	afero.WriteFile(overlayFs, "overlay.txt", []byte("overlay data"), 0644)

	// 3. Union FS (CopyOnWrite)
	unionFs := afero.NewCopyOnWriteFs(baseFs, overlayFs)

	t.Run("Read from base", func(t *testing.T) {
		// Should read "dir/nested.txt" which only exists in the base
		content, err := afero.ReadFile(unionFs, "dir/nested.txt")
		if err != nil {
			t.Fatalf("ReadFile failed: %v", err)
		}
		if string(content) != "nested content" {
			t.Errorf("Expected 'nested content', got '%s'", string(content))
		}
	})

	t.Run("Write new file", func(t *testing.T) {
		err := afero.WriteFile(unionFs, "new_union.txt", []byte("union data"), 0644)
		if err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}

		// Verify it exists in the overlay, not the base
		exists, _ := afero.Exists(overlayFs, "new_union.txt")
		if !exists {
			t.Error("New file should exist in overlay")
		}
		exists, _ = afero.Exists(baseFs, "new_union.txt")
		if exists {
			t.Error("New file should not exist in base")
		}
	})

	t.Run("Overwrite base file (Copy-on-Write)", func(t *testing.T) {
		// Modify file.txt, triggering copy-on-write
		err := afero.WriteFile(unionFs, "file.txt", []byte("modified"), 0644)
		if err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}

		// Base should remain unchanged
		content, _ := afero.ReadFile(baseFs, "file.txt")
		if string(content) != "hello world" {
			t.Errorf("Base FS content changed. Expected 'hello world', got '%s'", string(content))
		}

		// Overlay and Union should have the new content
		content, _ = afero.ReadFile(overlayFs, "file.txt")
		if string(content) != "modified" {
			t.Errorf("Overlay FS content incorrect. Expected 'modified', got '%s'", string(content))
		}
	})
}

// TestFSTestCompliance uses the standard library's fstest.TestFS.
func TestBridgeIOFS_FSTestCompliance(t *testing.T) {
	afs := afero.NewBridgeIOFS(setupTestBridgeFS())

	// Afero provides an adapter to go from afero.Fs back to io/fs.FS (afero.NewIOFS)
	// This tests the round-trip (io/fs -> BridgeIOFS -> IOFS adapter) ensuring compliance.
	iofs := afero.NewIOFS(afs)

	// Run the standard compliance test suite
	err := fstest.TestFS(iofs, "file.txt", "dir/nested.txt", "empty_dir")
	if err != nil {
		t.Errorf("BridgeIOFS failed standard io/fs compliance tests: %v", err)
	}
}

// --- Advanced Interface Tests (Seek, ReadAt) ---
// We need mocks because fstest.MapFS does not guarantee Seek or ReadAt support.

// BridgeMockFileInfo implements fs.FileInfo.
type BridgeMockFileInfo struct {
	name string
	size int64
	mode fs.FileMode
}

func (m BridgeMockFileInfo) Name() string       { return m.name }
func (m BridgeMockFileInfo) Size() int64        { return m.size }
func (m BridgeMockFileInfo) Mode() fs.FileMode  { return m.mode }
func (m BridgeMockFileInfo) ModTime() time.Time { return time.Now() }
func (m BridgeMockFileInfo) IsDir() bool        { return m.mode.IsDir() }
func (m BridgeMockFileInfo) Sys() interface{}   { return nil }

// BridgeSeekableFile implements fs.File, io.Seeker, and io.ReaderAt.
type BridgeSeekableFile struct {
	data   []byte
	name   string
	offset int64
}

func (sf *BridgeSeekableFile) Stat() (fs.FileInfo, error) {
	return BridgeMockFileInfo{name: sf.name, size: int64(len(sf.data)), mode: 0444}, nil
}
func (sf *BridgeSeekableFile) Close() error { return nil }
func (sf *BridgeSeekableFile) Read(p []byte) (int, error) {
	if sf.offset >= int64(len(sf.data)) {
		return 0, io.EOF
	}
	n := copy(p, sf.data[sf.offset:])
	sf.offset += int64(n)
	return n, nil
}
func (sf *BridgeSeekableFile) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = sf.offset + offset
	case io.SeekEnd:
		newOffset = int64(len(sf.data)) + offset
	}
	if newOffset < 0 {
		return 0, errors.New("invalid seek offset")
	}
	sf.offset = newOffset
	return sf.offset, nil
}
func (sf *BridgeSeekableFile) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, errors.New("invalid offset")
	}
	if off >= int64(len(sf.data)) {
		return 0, io.EOF
	}
	n := copy(p, sf.data[off:])
	if n < len(p) {
		// If we copied less than requested, it means we hit the end.
		return n, io.EOF
	}
	return n, nil
}

// BridgeSeekableFS implements fs.FS and returns a BridgeSeekableFile.
type BridgeSeekableFS struct{ file *BridgeSeekableFile }

func (sfs *BridgeSeekableFS) Open(name string) (fs.File, error) {
	// io/fs paths use forward slashes and dot cleaning.
	if name == sfs.file.name {
		sfs.file.offset = 0 // Reset offset on open
		return sfs.file, nil
	}
	return nil, fs.ErrNotExist
}

func TestBridgeIOFS_InterfacesSupported(t *testing.T) {
	// Test if Seek and ReadAt are correctly passed through if the underlying io/fs.File supports them.
	sfs := &BridgeSeekableFS{
		file: &BridgeSeekableFile{
			data: []byte("0123456789"),
			name: "data.bin",
		},
	}

	afs := afero.NewBridgeIOFS(sfs)
	f, err := afs.Open("data.bin")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer f.Close()

	// Run sequentially as the file offset state is shared.
	t.Run("Seek", func(t *testing.T) {
		pos, err := f.Seek(5, io.SeekStart)
		if err != nil {
			t.Fatalf("Seek failed: %v", err)
		}
		if pos != 5 {
			t.Errorf("Expected position 5, got %d", pos)
		}

		buf := make([]byte, 3)
		_, err = f.Read(buf)
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}
		if string(buf) != "567" {
			t.Errorf("Expected '567', got '%s'", string(buf))
		}
		// Current position is now 8
	})

	t.Run("ReadAt", func(t *testing.T) {
		buf := make([]byte, 4)
		// ReadAt should not affect the current seek position (which is 8 from the previous test)
		n, err := f.ReadAt(buf, 2)
		if err != nil && err != io.EOF {
			t.Fatalf("ReadAt failed: %v", err)
		}
		if n != 4 {
			t.Errorf("Expected to read 4 bytes, got %d", n)
		}
		if string(buf) != "2345" {
			t.Errorf("Expected '2345', got '%s'", string(buf))
		}

		// Verify seek position is unchanged (should still be 8)
		buf = make([]byte, 2)
		n, err = f.Read(buf)
		// We expect EOF or nil depending on how exactly the underlying Read handles the end.
		if err != nil && err != io.EOF {
			t.Fatalf("Read failed: %v", err)
		}
		if n != 2 {
			t.Errorf("Expected to read 2 bytes, got %d", n)
		}
		if string(buf) != "89" {
			t.Errorf("Expected '89', got '%s'", string(buf))
		}
	})
}

func TestBridgeIOFS_InterfacesNotSupported(t *testing.T) {
	// fstest.MapFS files generally do not implement Seek or ReadAt
	afs := afero.NewBridgeIOFS(setupTestBridgeFS())

	f, err := afs.Open("file.txt")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer f.Close()

	t.Run("Seek Fails", func(t *testing.T) {
		_, err = f.Seek(5, io.SeekStart)
		// Check that the specific Afero error is returned
		if !errors.Is(err, afero.ErrNoSeek) {
			t.Errorf("Expected afero.ErrNoSeek, got %v", err)
		}
		// Check that it is wrapped in PathError
		var pe *os.PathError
		if !errors.As(err, &pe) {
			t.Errorf("Expected error to be a PathError, got %T", err)
		}
	})

	t.Run("ReadAt Fails", func(t *testing.T) {
		_, err = f.ReadAt([]byte{1}, 5)
		// Check that the specific Afero error is returned
		if !errors.Is(err, afero.ErrNoReadAt) {
			t.Errorf("Expected afero.ErrNoReadAt, got %v", err)
		}
		// Check that it is wrapped in PathError
		var pe *os.PathError
		if !errors.As(err, &pe) {
			t.Errorf("Expected error to be a PathError, got %T", err)
		}
	})
}
