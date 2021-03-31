// Copyright © 2021 Vasily Ovchinnikov <vasily@remerge.io>.
//
// Most of the tests are "derived" from the Afero's own tarfs implementation.
// Write-oriented tests and/or checks have been added on top of that

package afero

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/spf13/afero/gcsfs"

	"cloud.google.com/go/storage"
	"github.com/googleapis/google-cloud-go-testing/storage/stiface"
)

const (
	testBytes = 8
	dirSize   = 42
)

var files = []struct {
	name            string
	exists          bool
	isdir           bool
	size            int64
	content         string
	offset          int64
	contentAtOffset string
}{
	{"", true, true, dirSize, "", 0, ""}, // this is NOT a valid path for GCS, so we do some magic here
	{"sub", true, true, dirSize, "", 0, ""},
	{"sub/testDir2", true, true, dirSize, "", 0, ""},
	{"sub/testDir2/testFile", true, false, 8 * 1024, "c", 4 * 1024, "d"},
	{"testFile", true, false, 12 * 1024, "a", 7 * 1024, "b"},
	{"testDir1/testFile", true, false, 3 * 512, "b", 512, "c"},

	{"nonExisting", false, false, dirSize, "", 0, ""},
}

var dirs = []struct {
	name     string
	children []string
}{
	{"", []string{"sub", "testDir1", "testFile"}},
	{"sub", []string{"testDir2"}},
	{"sub/testDir2", []string{"testFile"}},
	{"testDir1", []string{"testFile"}},
}

var gcsAfs *Afero

func TestMain(m *testing.M) {
	ctx := context.Background()
	var err error

	// Check if GOOGLE_APPLICATION_CREDENTIALS are present. If not, then a fake service account
	// would be used: https://github.com/google/oauth2l/blob/master/integration/fixtures/fake-service-account.json
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		var fakeCredentialsAbsPath string
		fakeCredentialsAbsPath, err = filepath.Abs("gcs-fake-service-account.json")
		if err != nil {
			fmt.Print(err)
			os.Exit(1)
		}

		err = os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", fakeCredentialsAbsPath)
		if err != nil {
			fmt.Print(err)
			os.Exit(1)
		}

		// reset it after the run
		defer func() {
			err = os.Remove("GOOGLE_APPLICATION_CREDENTIALS")
			if err != nil {
				// it's worth printing it out explicitly, since it might have implications further down the road
				fmt.Print("failed to clear fake GOOGLE_APPLICATION_CREDENTIALS", err)
			}
		}()
	}

	var c *storage.Client
	c, err = storage.NewClient(ctx)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	client := stiface.AdaptClient(c)

	// This block is mocking the client for the sake of isolated testing
	mockClient := newClientMock()
	mockClient.Client = client

	bucket := mockClient.Bucket("a-test-bucket")

	// If you want to run the test suite on a LIVE bucket, comment the previous
	// block and uncomment the line below and put your bucket name there.
	// Keep in mind, that GCS will likely rate limit you, so it would be impossible
	// to run the entire suite at once, only test by test.
	//bucket := client.Bucket("a-test-bucket")

	gcsAfs = &Afero{Fs: &GcsFs{gcsfs.NewGcsFs(ctx, bucket)}}

	// defer here to assure our Env cleanup happens, if the mock was used
	defer os.Exit(m.Run())
}

func createFiles(t *testing.T) {
	t.Helper()
	var err error

	// the files have to be created first
	for _, f := range files {
		if !f.isdir && f.exists {
			var freshFile File
			freshFile, err = gcsAfs.Create(f.name)
			if err != nil {
				t.Fatalf("failed to create a file \"%s\": %s", f.name, err)
			}

			var written int
			var totalWritten int64
			for totalWritten < f.size {
				if totalWritten < f.offset {
					writeBuf := []byte(strings.Repeat(f.content, int(f.offset)))
					written, err = freshFile.WriteAt(writeBuf, totalWritten)
				} else {
					writeBuf := []byte(strings.Repeat(f.contentAtOffset, int(f.size-f.offset)))
					written, err = freshFile.WriteAt(writeBuf, totalWritten)
				}
				if err != nil {
					t.Fatalf("failed to write a file \"%s\": %s", f.name, err)
				}

				totalWritten += int64(written)
			}

			err = freshFile.Close()
			if err != nil {
				t.Fatalf("failed to close a file \"%s\": %s", f.name, err)
			}
		}
	}
}

func removeFiles(t *testing.T) {
	t.Helper()
	var err error

	// the files have to be created first
	for _, f := range files {
		if !f.isdir && f.exists {
			err = gcsAfs.Remove(f.name)
			if err != nil && err == syscall.ENOENT {
				t.Errorf("failed to remove file \"%s\": %s", f.name, err)
			}
		}
	}
}

func TestFsOpen(t *testing.T) {
	createFiles(t)
	defer removeFiles(t)

	for _, f := range files {
		file, err := gcsAfs.Open(f.name)
		if (err == nil) != f.exists {
			t.Errorf("%v exists = %v, but got err = %v", f.name, f.exists, err)
		}

		if !f.exists {
			continue
		}
		if err != nil {
			t.Fatalf("%v: %v", f.name, err)
		}

		if file.Name() != filepath.FromSlash(f.name) {
			t.Errorf("Name(), got %v, expected %v", file.Name(), filepath.FromSlash(f.name))
		}

		s, err := file.Stat()
		if err != nil {
			t.Fatalf("stat %v: got error '%v'", file.Name(), err)
		}

		if isdir := s.IsDir(); isdir != f.isdir {
			t.Errorf("%v directory, got: %v, expected: %v", file.Name(), isdir, f.isdir)
		}

		if size := s.Size(); size != f.size {
			t.Errorf("%v size, got: %v, expected: %v", file.Name(), size, f.size)
		}
	}
}

func TestRead(t *testing.T) {
	createFiles(t)
	defer removeFiles(t)

	for _, f := range files {
		if !f.exists {
			continue
		}

		file, err := gcsAfs.Open(f.name)
		if err != nil {
			t.Fatalf("opening %v: %v", f.name, err)
		}

		buf := make([]byte, 8)
		n, err := file.Read(buf)
		if err != nil {
			if f.isdir && (err != syscall.EISDIR) {
				t.Errorf("%v got error %v, expected EISDIR", f.name, err)
			} else if !f.isdir {
				t.Errorf("%v: %v", f.name, err)
			}
		} else if n != 8 {
			t.Errorf("%v: got %d read bytes, expected 8", f.name, n)
		} else if string(buf) != strings.Repeat(f.content, testBytes) {
			t.Errorf("%v: got <%s>, expected <%s>", f.name, f.content, string(buf))
		}

	}
}

func TestGcsReadAt(t *testing.T) {
	createFiles(t)
	defer removeFiles(t)

	for _, f := range files {
		if !f.exists {
			continue
		}

		file, err := gcsAfs.Open(f.name)
		if err != nil {
			t.Fatalf("opening %v: %v", f.name, err)
		}

		buf := make([]byte, testBytes)
		n, err := file.ReadAt(buf, f.offset-testBytes/2)
		if err != nil {
			if f.isdir && (err != syscall.EISDIR) {
				t.Errorf("%v got error %v, expected EISDIR", f.name, err)
			} else if !f.isdir {
				t.Errorf("%v: %v", f.name, err)
			}
		} else if n != 8 {
			t.Errorf("%v: got %d read bytes, expected 8", f.name, n)
		} else if string(buf) != strings.Repeat(f.content, testBytes/2)+strings.Repeat(f.contentAtOffset, testBytes/2) {
			t.Errorf("%v: got <%s>, expected <%s>", f.name, f.contentAtOffset, string(buf))
		}

	}
}

func TestGcsSeek(t *testing.T) {
	createFiles(t)
	defer removeFiles(t)

	for _, f := range files {
		if !f.exists {
			continue
		}

		file, err := gcsAfs.Open(f.name)
		if err != nil {
			t.Fatalf("opening %v: %v", f.name, err)
		}

		var tests = []struct {
			offIn  int64
			whence int
			offOut int64
		}{
			{0, io.SeekStart, 0},
			{10, io.SeekStart, 10},
			{1, io.SeekCurrent, 11},
			{10, io.SeekCurrent, 21},
			{0, io.SeekEnd, f.size},
			{-1, io.SeekEnd, f.size - 1},
		}

		for _, s := range tests {
			n, err := file.Seek(s.offIn, s.whence)
			if err != nil {
				if f.isdir && err == syscall.EISDIR {
					continue
				}

				t.Errorf("%v: %v", f.name, err)
			}

			if n != s.offOut {
				t.Errorf("%v: (off: %v, whence: %v): got %v, expected %v", f.name, s.offIn, s.whence, n, s.offOut)
			}
		}

	}
}

func TestName(t *testing.T) {
	createFiles(t)
	defer removeFiles(t)

	for _, f := range files {
		if !f.exists {
			continue
		}

		file, err := gcsAfs.Open(f.name)
		if err != nil {
			t.Fatalf("opening %v: %v", f.name, err)
		}

		n := file.Name()
		if n != filepath.FromSlash(f.name) {
			t.Errorf("got: %v, expected: %v", n, filepath.FromSlash(f.name))
		}

	}
}

func TestClose(t *testing.T) {
	createFiles(t)
	defer removeFiles(t)

	for _, f := range files {
		if !f.exists {
			continue
		}

		file, err := gcsAfs.Open(f.name)
		if err != nil {
			t.Fatalf("opening %v: %v", f.name, err)
		}

		err = file.Close()
		if err != nil {
			t.Errorf("%v: %v", f.name, err)
		}

		err = file.Close()
		if err == nil {
			t.Errorf("%v: closing twice should return an error", f.name)
		}

		buf := make([]byte, 8)
		n, err := file.Read(buf)
		if n != 0 || err == nil {
			t.Errorf("%v: could read from a closed file", f.name)
		}

		n, err = file.ReadAt(buf, 256)
		if n != 0 || err == nil {
			t.Errorf("%v: could readAt from a closed file", f.name)
		}

		off, err := file.Seek(0, io.SeekStart)
		if off != 0 || err == nil {
			t.Errorf("%v: could seek from a closed file", f.name)
		}
	}
}

func TestGcsOpenFile(t *testing.T) {
	createFiles(t)
	defer removeFiles(t)

	for _, f := range files {
		file, err := gcsAfs.OpenFile(f.name, os.O_RDONLY, 0400)
		if !f.exists {
			if !errors.Is(err, syscall.ENOENT) {
				t.Errorf("%v: got %v, expected%v", f.name, err, syscall.ENOENT)
			}

			continue
		}

		if err != nil {
			t.Fatalf("%v: %v", f.name, err)
		}

		err = file.Close()
		if err != nil {
			t.Fatalf("failed to close a file \"%s\": %s", f.name, err)
		}

		file, err = gcsAfs.OpenFile(f.name, os.O_CREATE, 0600)
		if !errors.Is(err, syscall.EPERM) {
			t.Errorf("%v: open for write: got %v, expected %v", f.name, err, syscall.EPERM)
		}

	}
}

func TestFsStat(t *testing.T) {
	createFiles(t)
	defer removeFiles(t)

	for _, f := range files {
		fi, err := gcsAfs.Stat(f.name)
		if !f.exists {
			if !errors.Is(err, syscall.ENOENT) {
				t.Errorf("%v: got %v, expected%v", f.name, err, syscall.ENOENT)
			}

			continue
		}

		if err != nil {
			t.Fatalf("stat %v: got error '%v'", f.name, err)
		}

		if isdir := fi.IsDir(); isdir != f.isdir {
			t.Errorf("%v directory, got: %v, expected: %v", f.name, isdir, f.isdir)
		}

		if size := fi.Size(); size != f.size {
			t.Errorf("%v size, got: %v, expected: %v", f.name, size, f.size)
		}
	}
}

func TestGcsReaddir(t *testing.T) {
	createFiles(t)
	defer removeFiles(t)

	for _, d := range dirs {
		dir, err := gcsAfs.Open(d.name)
		if err != nil {
			t.Fatal(err)
		}

		fi, err := dir.Readdir(0)
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, f := range fi {
			names = append(names, f.Name())
		}

		if !reflect.DeepEqual(names, d.children) {
			t.Errorf("%v: children, got '%v', expected '%v'", d.name, names, d.children)
		}

		fi, err = dir.Readdir(1)
		if err != nil {
			t.Fatal(err)
		}

		names = []string{}
		for _, f := range fi {
			names = append(names, f.Name())
		}

		if !reflect.DeepEqual(names, d.children[0:1]) {
			t.Errorf("%v: children, got '%v', expected '%v'", d.name, names, d.children[0:1])
		}
	}

	dir, err := gcsAfs.Open("testFile")
	if err != nil {
		t.Fatal(err)
	}

	_, err = dir.Readdir(-1)
	if err != syscall.ENOTDIR {
		t.Fatal("Expected error")
	}
}

func TestGcsReaddirnames(t *testing.T) {
	createFiles(t)
	defer removeFiles(t)

	for _, d := range dirs {
		dir, err := gcsAfs.Open(d.name)
		if err != nil {
			t.Fatal(err)
		}

		names, err := dir.Readdirnames(0)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(names, d.children) {
			t.Errorf("%v: children, got '%v', expected '%v'", d.name, names, d.children)
		}

		names, err = dir.Readdirnames(1)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(names, d.children[0:1]) {
			t.Errorf("%v: children, got '%v', expected '%v'", d.name, names, d.children[0:1])
		}
	}

	dir, err := gcsAfs.Open("testFile")
	if err != nil {
		t.Fatal(err)
	}

	_, err = dir.Readdir(-1)
	if err != syscall.ENOTDIR {
		t.Fatal("Expected error")
	}
}

func TestGcsGlob(t *testing.T) {
	createFiles(t)
	defer removeFiles(t)

	for _, s := range []struct {
		glob    string
		entries []string
	}{
		{filepath.FromSlash("/*"), []string{filepath.FromSlash("/sub"), filepath.FromSlash("/testDir1"), filepath.FromSlash("/testFile")}},
		{filepath.FromSlash("*"), []string{filepath.FromSlash("sub"), filepath.FromSlash("testDir1"), filepath.FromSlash("testFile")}},
		{filepath.FromSlash("/sub/*"), []string{filepath.FromSlash("/sub/testDir2")}},
		{filepath.FromSlash("/sub/testDir2/*"), []string{filepath.FromSlash("/sub/testDir2/testFile")}},
		{filepath.FromSlash("/testDir1/*"), []string{filepath.FromSlash("/testDir1/testFile")}},
		{filepath.FromSlash("sub/*"), []string{filepath.FromSlash("sub/testDir2")}},
		{filepath.FromSlash("sub/testDir2/*"), []string{filepath.FromSlash("sub/testDir2/testFile")}},
		{filepath.FromSlash("testDir1/*"), []string{filepath.FromSlash("testDir1/testFile")}},
	} {
		entries, err := Glob(gcsAfs.Fs, s.glob)
		if err != nil {
			t.Error(err)
		}
		if reflect.DeepEqual(entries, s.entries) {
			t.Logf("glob: %s: glob ok", s.glob)
		} else {
			t.Errorf("glob: %s: got %#v, expected %#v", s.glob, entries, s.entries)
		}
	}
}

func TestMkdir(t *testing.T) {
	dirName := "/a-test-dir"
	var err error

	err = gcsAfs.Mkdir(dirName, 0755)
	if err != nil {
		t.Fatal("failed to create a folder with error", err)
	}

	info, err := gcsAfs.Stat(dirName)
	if err != nil {
		t.Fatal("failed to get info", err)
	}
	if !info.IsDir() {
		t.Fatalf("%s: not a dir", dirName)
	}
	if !info.Mode().IsDir() {
		t.Errorf("%s: mode is not directory", dirName)
	}

	if info.Mode() != os.ModeDir|0755 {
		t.Errorf("%s: wrong permissions, expected drwxr-xr-x, got %s", dirName, info.Mode())
	}

	err = gcsAfs.Remove(dirName)
	if err != nil {
		t.Fatalf("could not delete the folder %s after the test with error: %s", dirName, err)
	}
}

func TestMkdirAll(t *testing.T) {
	err := gcsAfs.MkdirAll("/a/b/c", 0755)
	if err != nil {
		t.Fatal(err)
	}

	info, err := gcsAfs.Stat("/a")
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsDir() {
		t.Error("/a: mode is not directory")
	}
	if info.Mode() != os.ModeDir|0755 {
		t.Errorf("/a: wrong permissions, expected drwxr-xr-x, got %s", info.Mode())
	}
	info, err = gcsAfs.Stat("/a/b")
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsDir() {
		t.Error("/a/b: mode is not directory")
	}
	if info.Mode() != os.ModeDir|0755 {
		t.Errorf("/a/b: wrong permissions, expected drwxr-xr-x, got %s", info.Mode())
	}
	info, err = gcsAfs.Stat("/a/b/c")
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsDir() {
		t.Error("/a/b/c: mode is not directory")
	}
	if info.Mode() != os.ModeDir|0755 {
		t.Errorf("/a/b/c: wrong permissions, expected drwxr-xr-x, got %s", info.Mode())
	}

	err = gcsAfs.RemoveAll("/a")
	if err != nil {
		t.Fatalf("failed to remove the folder /a with error: %s", err)
	}
}
