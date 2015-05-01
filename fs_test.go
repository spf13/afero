// Copyright © 2014 Steve Francia <spf@spf13.com>.
// Copyright 2009 The Go Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package afero

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

var dot = []string{
	"fs.go",
	"fs_test.go",
	"httpFs.go",
	"memfile.go",
	"memmap.go",
}

var testDir = "/tmp/fun"
var testName = "test.txt"
var Fss = []Fs{&MemMapFs{}, &OsFs{}}

//var Fss = []Fs{OsFs{}}

//Read with length 0 should not return EOF.
func TestRead0(t *testing.T) {
	for _, fs := range Fss {
		path := testDir + "/" + testName
		if err := fs.MkdirAll(testDir, 0777); err != nil {
			t.Fatal(fs.Name(), "unable to create dir", err)
		}

		f, err := fs.Create(path)
		if err != nil {
			t.Fatal(fs.Name(), "create failed:", err)
		}
		defer f.Close()
		f.WriteString("Lorem ipsum dolor sit amet, consectetur adipisicing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.")

		b := make([]byte, 0)
		n, err := f.Read(b)
		if n != 0 || err != nil {
			t.Errorf("%v: Read(0) = %d, %v, want 0, nil", fs.Name(), n, err)
		}
		f.Seek(0, 0)
		b = make([]byte, 100)
		n, err = f.Read(b)
		if n <= 0 || err != nil {
			t.Errorf("%v: Read(100) = %d, %v, want >0, nil", fs.Name(), n, err)
		}
	}
}

func TestMemFileRead(t *testing.T) {
	f := MemFileCreate("testfile")
	f.WriteString("abcd")
	f.Seek(0, 0)
	b := make([]byte, 8)
	n, err := f.Read(b)
	if n != 4 {
		t.Errorf("didn't read all bytes: %v %v %v", n, err, b)
	}
	if err != nil {
		t.Errorf("err is not nil: %v %v %v", n, err, b)
	}
	n, err = f.Read(b)
	if n != 0 {
		t.Errorf("read more bytes: %v %v %v", n, err, b)
	}
	if err != io.EOF {
		t.Errorf("error is not EOF: %v %v %v", n, err, b)
	}
}

func TestRename(t *testing.T) {
	for _, fs := range Fss {
		from, to := testDir+"/renamefrom", testDir+"/renameto"
		fs.Remove(to)              // Just in case.
		fs.MkdirAll(testDir, 0777) // Just in case.
		file, err := fs.Create(from)
		if err != nil {
			t.Fatalf("open %q failed: %v", to, err)
		}
		if err = file.Close(); err != nil {
			t.Errorf("close %q failed: %v", to, err)
		}
		err = fs.Rename(from, to)
		if err != nil {
			t.Fatalf("rename %q, %q failed: %v", to, from, err)
		}
		defer fs.Remove(to)
		_, err = fs.Stat(to)
		if err != nil {
			t.Errorf("stat %q failed: %v", to, err)
		}
	}
}

func TestTruncate(t *testing.T) {
	for _, fs := range Fss {
		f := newFile("TestTruncate", fs, t)
		defer fs.Remove(f.Name())
		defer f.Close()

		checkSize(t, f, 0)
		f.Write([]byte("hello, world\n"))
		checkSize(t, f, 13)
		f.Truncate(10)
		checkSize(t, f, 10)
		f.Truncate(1024)
		checkSize(t, f, 1024)
		f.Truncate(0)
		checkSize(t, f, 0)
		_, err := f.Write([]byte("surprise!"))
		if err == nil {
			checkSize(t, f, 13+9) // wrote at offset past where hello, world was.
		}
	}
}

func TestSeek(t *testing.T) {
	for _, fs := range Fss {
		f := newFile("TestSeek", fs, t)
		defer fs.Remove(f.Name())
		defer f.Close()

		const data = "hello, world\n"
		io.WriteString(f, data)

		type test struct {
			in     int64
			whence int
			out    int64
		}
		var tests = []test{
			{0, 1, int64(len(data))},
			{0, 0, 0},
			{5, 0, 5},
			{0, 2, int64(len(data))},
			{0, 0, 0},
			{-1, 2, int64(len(data)) - 1},
			{1 << 33, 0, 1 << 33},
			{1 << 33, 2, 1<<33 + int64(len(data))},
		}
		for i, tt := range tests {
			off, err := f.Seek(tt.in, tt.whence)
			if off != tt.out || err != nil {
				if e, ok := err.(*os.PathError); ok && e.Err == syscall.EINVAL && tt.out > 1<<32 {
					// Reiserfs rejects the big seeks.
					// http://code.google.com/p/go/issues/detail?id=91
					break
				}
				t.Errorf("#%d: Seek(%v, %v) = %v, %v want %v, nil", i, tt.in, tt.whence, off, err, tt.out)
			}
		}
	}
}

func TestReadAt(t *testing.T) {
	for _, fs := range Fss {
		f := newFile("TestReadAt", fs, t)
		defer fs.Remove(f.Name())
		defer f.Close()

		const data = "hello, world\n"
		io.WriteString(f, data)

		b := make([]byte, 5)
		n, err := f.ReadAt(b, 7)
		if err != nil || n != len(b) {
			t.Fatalf("ReadAt 7: %d, %v", n, err)
		}
		if string(b) != "world" {
			t.Fatalf("ReadAt 7: have %q want %q", string(b), "world")
		}
	}
}

func TestWriteAt(t *testing.T) {
	for _, fs := range Fss {
		f := newFile("TestWriteAt", fs, t)
		defer fs.Remove(f.Name())
		defer f.Close()

		const data = "hello, world\n"
		io.WriteString(f, data)

		n, err := f.WriteAt([]byte("WORLD"), 7)
		if err != nil || n != 5 {
			t.Fatalf("WriteAt 7: %d, %v", n, err)
		}

		f2, err := fs.Open(f.Name())
		defer f2.Close()
		buf := new(bytes.Buffer)
		buf.ReadFrom(f2)
		b := buf.Bytes()
		if err != nil {
			t.Fatalf("%v: ReadFile %s: %v", fs.Name(), f.Name(), err)
		}
		if string(b) != "hello, WORLD\n" {
			t.Fatalf("after write: have %q want %q", string(b), "hello, WORLD\n")
		}

	}
}

//func TestReaddirnames(t *testing.T) {
//for _, fs := range Fss {
//testReaddirnames(fs, ".", dot, t)
////testReaddirnames(sysdir.name, fs, sysdir.files, t)
//}
//}

//func TestReaddir(t *testing.T) {
//for _, fs := range Fss {
//testReaddir(fs, ".", dot, t)
////testReaddir(sysdir.name, fs, sysdir.files, t)
//}
//}

func newFile(testName string, fs Fs, t *testing.T) (f File) {
	// Use a local file system, not NFS.
	// On Unix, override $TMPDIR in case the user
	// has it set to an NFS-mounted directory.
	dir := ""
	if runtime.GOOS != "windows" {
		dir = "/tmp"
	}
	fs.MkdirAll(dir, 0777)
	f, err := fs.Create(path.Join(dir, testName))
	if err != nil {
		t.Fatalf("%v: open %s: %s", fs.Name(), testName, err)
	}
	return f
}

func writeFile(t *testing.T, fs Fs, fname string, flag int, text string) string {
	f, err := fs.OpenFile(fname, flag, 0666)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	n, err := io.WriteString(f, text)
	if err != nil {
		t.Fatalf("WriteString: %d, %v", n, err)
	}
	f.Close()
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	return string(data)
}

func testReaddirnames(fs Fs, dir string, contents []string, t *testing.T) {
	file, err := fs.Open(dir)
	if err != nil {
		t.Fatalf("open %q failed: %v", dir, err)
	}
	defer file.Close()
	s, err2 := file.Readdirnames(-1)
	if err2 != nil {
		t.Fatalf("readdirnames %q failed: %v", dir, err2)
	}
	for _, m := range contents {
		found := false
		for _, n := range s {
			if n == "." || n == ".." {
				t.Errorf("got %s in directory", n)
			}
			if equal(m, n) {
				if found {
					t.Error("present twice:", m)
				}
				found = true
			}
		}
		if !found {
			t.Error("could not find", m)
		}
	}
}

func testReaddir(fs Fs, dir string, contents []string, t *testing.T) {
	file, err := fs.Open(dir)
	if err != nil {
		t.Fatalf("open %q failed: %v", dir, err)
	}
	defer file.Close()
	s, err2 := file.Readdir(-1)
	if err2 != nil {
		t.Fatalf("readdir %q failed: %v", dir, err2)
	}
	for _, m := range contents {
		found := false
		for _, n := range s {
			if equal(m, n.Name()) {
				if found {
					t.Error("present twice:", m)
				}
				found = true
			}
		}
		if !found {
			t.Error("could not find", m)
		}
	}
}

func equal(name1, name2 string) (r bool) {
	switch runtime.GOOS {
	case "windows":
		r = strings.ToLower(name1) == strings.ToLower(name2)
	default:
		r = name1 == name2
	}
	return
}

func checkSize(t *testing.T, f File, size int64) {
	dir, err := f.Stat()
	if err != nil {
		t.Fatalf("Stat %q (looking for size %d): %s", f.Name(), size, err)
	}
	if dir.Size() != size {
		t.Errorf("Stat %q: size %d want %d", f.Name(), dir.Size(), size)
	}
}
