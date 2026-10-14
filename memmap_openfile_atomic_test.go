package afero

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestMemMapOpenFileExclusiveConcurrent(t *testing.T) {
	const workers = 32
	for round := 0; round < 32; round++ {
		fs := NewMemMapFs()
		start := make(chan struct{})
		results := make(chan error, workers)
		var ready sync.WaitGroup
		ready.Add(workers)
		for i := 0; i < workers; i++ {
			go func(i int) {
				name := "shared"
				if i%2 == 0 {
					name = filepath.Join(FilePathSeparator, name)
				}
				ready.Done()
				<-start
				f, err := fs.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o640)
				if err == nil {
					err = f.Close()
				}
				results <- err
			}(i)
		}
		ready.Wait()
		close(start)
		succeeded := 0
		for i := 0; i < workers; i++ {
			err := <-results
			if err == nil {
				succeeded++
			} else if !os.IsExist(err) {
				t.Errorf("round %d: unexpected open error: %v", round, err)
			}
		}
		if succeeded != 1 {
			t.Fatalf("round %d: exclusive creation succeeded %d times; want 1", round, succeeded)
		}
	}
}

func TestMemMapOpenFileCreateConcurrentSharedFile(t *testing.T) {
	const workers = 32
	for round := 0; round < 32; round++ {
		fs := NewMemMapFs()
		start := make(chan struct{})
		type result struct {
			file File
			err  error
		}
		results := make(chan result, workers)
		var ready sync.WaitGroup
		ready.Add(workers)
		for i := 0; i < workers; i++ {
			go func() {
				ready.Done()
				<-start
				f, err := fs.OpenFile("shared", os.O_CREATE|os.O_RDWR, 0o640)
				results <- result{f, err}
			}()
		}
		ready.Wait()
		close(start)
		want := make([]byte, workers)
		for i := 0; i < workers; i++ {
			r := <-results
			want[i] = byte(i + 1)
			if r.err != nil {
				t.Errorf("round %d: unexpected open error: %v", round, r.err)
				continue
			}
			// Write only after this handle is returned. Other opens without O_TRUNC
			// must keep the same file, including bytes already written through it.
			if _, err := r.file.WriteAt(want[i:i+1], int64(i)); err != nil {
				t.Errorf("write: %v", err)
			}
			if err := r.file.Close(); err != nil {
				t.Errorf("close: %v", err)
			}
		}
		got, err := ReadFile(fs, "shared")
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("round %d: concurrent creates lost bytes: got %v, want %v", round, got, want)
		}
	}
}

func TestMemMapOpenFileCreatePreservesExistingFile(t *testing.T) {
	fs := NewMemMapFs()
	f, err := fs.OpenFile("shared", os.O_CREATE|os.O_RDWR, 0o640)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("keep"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	f, err = fs.OpenFile("shared", os.O_CREATE|os.O_RDWR, 0o400)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFile(fs, "shared")
	if err != nil || string(got) != "keep" {
		t.Fatalf("existing contents: %q, %v", got, err)
	}
	info, err := fs.Stat("shared")
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("existing mode changed to %o", info.Mode().Perm())
	}
}
