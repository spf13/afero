// package test provides generic tests of Fs objects
package test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"github.com/spf13/afero"
)

var testName = "test.txt"

func mustTempDir(fs afero.Fs) string {
	name, err := afero.TempDir(fs, "", "afero")
	if err != nil {
		panic(fmt.Sprint("unable to work with test dir", err))
	}
	return name
}

func mustTempFile(fs afero.Fs) afero.File {
	x, err := afero.TempFile(fs, "", "afero")
	if err != nil {
		panic(fmt.Sprint("unable to work with temp file", err))
	}
	return x
}

func Read0(t *testing.T, fs afero.Fs) {
	f := mustTempFile(fs)
	defer f.Close()
	defer fs.Remove(f.Name())
	f.WriteString("Lorem ipsum dolor sit amet, consectetur adipisicing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.")

	var b []byte
	// b := make([]byte, 0)
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

func OpenFile(t *testing.T, fs afero.Fs) {
	tmp := mustTempDir(fs)
	defer fs.RemoveAll(tmp)

	path := filepath.Join(tmp, testName)
	f, err := fs.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		t.Fatal(fs.Name(), "OpenFile (O_CREATE) failed:", err)
	}
	io.WriteString(f, "initial")
	f.Close()

	f, err = fs.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		t.Fatal(fs.Name(), "OpenFile (O_APPEND) failed:", err)
	}
	io.WriteString(f, "|append")
	f.Close()

	f, err = fs.OpenFile(path, os.O_RDONLY, 0600)
	contents, _ := ioutil.ReadAll(f)
	expectedContents := "initial|append"
	if string(contents) != expectedContents {
		t.Errorf("%v: appending, expected '%v', got: '%v'", fs.Name(), expectedContents, string(contents))
	}
	f.Close()

	f, err = fs.OpenFile(path, os.O_RDWR|os.O_TRUNC, 0600)
	if err != nil {
		t.Fatal(fs.Name(), "OpenFile (O_TRUNC) failed:", err)
	}
	contents, _ = ioutil.ReadAll(f)
	if string(contents) != "" {
		t.Errorf("%v: expected truncated file, got: '%v'", fs.Name(), string(contents))
	}
	f.Close()
}

func Create(t *testing.T, fs afero.Fs) {
	tmp := mustTempDir(fs)
	defer fs.RemoveAll(tmp)

	path := filepath.Join(tmp, testName)
	f, err := fs.Create(path)
	if err != nil {
		f.Close()
		t.Fatal(fs.Name(), "Create failed:", err)
	}
	io.WriteString(f, "initial")
	f.Close()

	f, err = fs.Create(path)
	if err != nil {
		f.Close()
		t.Fatal(fs.Name(), "Create failed:", err)
	}
	secondContent := "second create"
	io.WriteString(f, secondContent)
	f.Close()

	f, err = fs.Open(path)
	if err != nil {
		f.Close()
		t.Fatal(fs.Name(), "Open failed:", err)
	}
	buf, err := afero.ReadAll(f)
	if err != nil {
		f.Close()
		t.Fatal(fs.Name(), "ReadAll failed:", err)
	}
	if string(buf) != secondContent {
		f.Close()
		t.Fatal(fs.Name(), "Content should be", "\""+secondContent+"\" but is \""+string(buf)+"\"")
	}
	f.Close()
}

func Rename(t *testing.T, fs afero.Fs) {
	tDir := mustTempDir(fs)
	defer fs.RemoveAll(tDir)

	from := filepath.Join(tDir, "/renamefrom")
	to := filepath.Join(tDir, "/renameto")
	exists := filepath.Join(tDir, "/renameexists")
	file, err := fs.Create(from)
	if err != nil {
		t.Fatalf("%s: open %q failed: %v", fs.Name(), to, err)
	}
	if err = file.Close(); err != nil {
		t.Errorf("%s: close %q failed: %v", fs.Name(), to, err)
	}
	file, err = fs.Create(exists)
	if err != nil {
		t.Fatalf("%s: open %q failed: %v", fs.Name(), to, err)
	}
	if err = file.Close(); err != nil {
		t.Errorf("%s: close %q failed: %v", fs.Name(), to, err)
	}
	err = fs.Rename(from, to)
	if err != nil {
		t.Fatalf("%s: rename %q, %q failed: %v", fs.Name(), to, from, err)
	}
	file, err = fs.Create(from)
	if err != nil {
		t.Fatalf("%s: open %q failed: %v", fs.Name(), to, err)
	}
	if err = file.Close(); err != nil {
		t.Errorf("%s: close %q failed: %v", fs.Name(), to, err)
	}
	err = fs.Rename(from, exists)
	if err != nil {
		t.Errorf("%s: rename %q, %q failed: %v", fs.Name(), exists, from, err)
	}
	names, err := afero.ReadDirNames(fs, tDir)
	if err != nil {
		t.Errorf("%s: readDirNames error: %v", fs.Name(), err)
	}
	found := false
	for _, e := range names {
		if e == "renamefrom" {
			t.Error("File is still called renamefrom")
		}
		if e == "renameto" {
			found = true
		}
	}
	if !found {
		t.Error("File was not renamed to renameto")
	}

	_, err = fs.Stat(to)
	if err != nil {
		t.Errorf("%s: stat %q failed: %v", fs.Name(), to, err)
	}
}

func Remove(t *testing.T, fs afero.Fs) {
	x, err := afero.TempFile(fs, "", "afero")
	if err != nil {
		t.Error(fmt.Sprint("unable to work with temp file", err))
	}

	path := x.Name()
	x.Close()

	tDir := filepath.Dir(path)

	err = fs.Remove(path)
	if err != nil {
		t.Fatalf("%v: Remove() failed: %v", fs.Name(), err)
	}

	_, err = fs.Stat(path)
	if !os.IsNotExist(err) {
		t.Fatalf("%v: Remove() didn't remove file", fs.Name())
	}

	// Deleting non-existent file should raise error
	err = fs.Remove(path)
	if !os.IsNotExist(err) {
		t.Errorf("%v: Remove() didn't raise error for non-existent file", fs.Name())
	}

	f, err := fs.Open(tDir)
	if err != nil {
		t.Error("TestDir should still exist:", err)
	}

	names, err := f.Readdirnames(-1)
	if err != nil {
		t.Error("Readdirnames failed:", err)
	}

	for _, e := range names {
		if e == testName {
			t.Error("File was not removed from parent directory")
		}
	}
}

func Truncate(t *testing.T, fs afero.Fs) {
	f := mustTempFile(fs)
	defer f.Close()
	defer fs.Remove(f.Name())

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

func ReadWriteSeek(t *testing.T, fs afero.Fs) {
	f := mustTempFile(fs)
	defer f.Close()
	defer fs.Remove(f.Name())

	const data = "hello, world\n"
	io.WriteString(f, data)

	type readInput struct {
		p []byte
	}
	type readOutput struct {
		p   []byte
		n   int
		err error
	}
	type readTest struct {
		input    readInput
		expected readOutput
	}
	type writeInput struct {
		p []byte
	}
	type writeOutput struct {
		n   int
		err error
	}
	type writeTest struct {
		input    writeInput
		expected writeOutput
	}
	type seekInput struct {
		offset int64
		whence int
	}
	type seekOutput struct {
		offset int64
		err    error
	}
	type seekTest struct {
		input    seekInput
		expected seekOutput
	}
	var tests = []interface{}{
		readTest{
			input:    readInput{make([]byte, 2)},
			expected: readOutput{make([]byte, 2), 0, io.EOF},
		},
		seekTest{
			input:    seekInput{-1, 1},
			expected: seekOutput{int64(len(data) - 1), nil},
		},
		readTest{
			input:    readInput{make([]byte, 2)},
			expected: readOutput{[]byte{'\n', uint8(0)}, 1, nil},
		},
		seekTest{
			input:    seekInput{0, 0},
			expected: seekOutput{0, nil},
		},
		readTest{
			input:    readInput{make([]byte, len(data))},
			expected: readOutput{[]byte(data), len(data), nil},
		},
		seekTest{
			input:    seekInput{0, 0},
			expected: seekOutput{0, nil},
		},
		writeTest{
			input:    writeInput{[]byte("c")},
			expected: writeOutput{1, nil},
		},
		readTest{
			input:    readInput{make([]byte, len(data)-1)},
			expected: readOutput{[]byte(data[1:]), len(data[1:]), nil},
		},
		seekTest{
			input:    seekInput{-1, 1},
			expected: seekOutput{int64(len(data) - 1), nil},
		},
		writeTest{
			input:    writeInput{[]byte("!\n")},
			expected: writeOutput{2, nil},
		},
		seekTest{
			input:    seekInput{0, 0},
			expected: seekOutput{0, nil},
		},
		readTest{
			input:    readInput{make([]byte, len(data)+1)},
			expected: readOutput{[]byte("cello, world!\n"), len("cello, world!\n"), nil},
		},
	}
	for idx, testi := range tests {
		switch test := testi.(type) {
		case readTest:
			n, err := f.Read(test.input.p)
			got := readOutput{
				p:   test.input.p,
				n:   n,
				err: err,
			}
			if !reflect.DeepEqual(got, test.expected) {
				t.Fatalf("read %d\nexpected %#v\ngot      %#v", idx, test.expected, got)
			}
		case writeTest:
			n, err := f.Write(test.input.p)
			got := writeOutput{
				n:   n,
				err: err,
			}
			if !reflect.DeepEqual(got, test.expected) {
				t.Fatalf("write %d\nexpected %#v\ngot     %#v", idx, test.expected, got)
			}
		case seekTest:
			offset, err := f.Seek(test.input.offset, test.input.whence)
			got := seekOutput{
				offset: offset,
				err:    err,
			}
			if !reflect.DeepEqual(got, test.expected) {
				t.Fatalf("seek %d\nexpected %#v\ngot     %#v", idx, test.expected, got)
			}
		}
	}
}

func Seek(t *testing.T, fs afero.Fs) {
	f := mustTempFile(fs)
	defer f.Close()
	defer fs.Remove(f.Name())

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

func ReadAt(t *testing.T, fs afero.Fs) {
	f := mustTempFile(fs)
	defer f.Close()
	defer fs.Remove(f.Name())

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

func WriteAt(t *testing.T, fs afero.Fs) {
	f := mustTempFile(fs)
	defer f.Close()
	defer fs.Remove(f.Name())

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

func setupTestFiles(t *testing.T, fs afero.Fs, path string) string {
	testSubDir := filepath.Join(path, "more", "subdirectories", "for", "testing", "we")
	err := fs.MkdirAll(testSubDir, 0700)
	if err != nil && !os.IsExist(err) {
		t.Fatal(err)
	}

	f, err := fs.Create(filepath.Join(testSubDir, "testfile1"))
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("Testfile 1 content")
	f.Close()

	f, err = fs.Create(filepath.Join(testSubDir, "testfile2"))
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("Testfile 2 content")
	f.Close()

	f, err = fs.Create(filepath.Join(testSubDir, "testfile3"))
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("Testfile 3 content")
	f.Close()

	f, err = fs.Create(filepath.Join(testSubDir, "testfile4"))
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("Testfile 4 content")
	f.Close()
	return testSubDir
}

func Readdirnames(t *testing.T, fs afero.Fs) {
	tmpDir := mustTempDir(fs)
	defer fs.RemoveAll(tmpDir)
	testSubDir := setupTestFiles(t, fs, tmpDir)
	tDir := filepath.Dir(testSubDir)

	root, err := fs.Open(tDir)
	if err != nil {
		t.Fatal(fs.Name(), tDir, err)
	}
	defer root.Close()

	namesRoot, err := root.Readdirnames(-1)
	if err != nil {
		t.Fatal(fs.Name(), namesRoot, err)
	}

	sub, err := fs.Open(testSubDir)
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Close()

	namesSub, err := sub.Readdirnames(-1)
	if err != nil {
		t.Fatal(fs.Name(), namesSub, err)
	}

	findNames(fs, t, tDir, testSubDir, namesRoot, namesSub)
}

func ReaddirSimple(t *testing.T, fs afero.Fs) {
	tmpDir := mustTempDir(fs)
	defer fs.RemoveAll(tmpDir)
	testSubDir := setupTestFiles(t, fs, tmpDir)
	tDir := filepath.Dir(testSubDir)

	root, err := fs.Open(tDir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	rootInfo, err := root.Readdir(1)
	if err != nil {
		t.Log(myFileInfo(rootInfo))
		t.Error(err)
	}

	rootInfo, err = root.Readdir(5)
	if err != io.EOF {
		t.Log(myFileInfo(rootInfo))
		t.Errorf("%s second readdir should have been io.EOF, got %s", root.Name(), err)
	}

	sub, err := fs.Open(testSubDir)
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Close()

	subInfo, err := sub.Readdir(5)
	if err != nil {
		t.Log(myFileInfo(subInfo))
		t.Error(err)
	}
}

type myFileInfo []os.FileInfo

func (m myFileInfo) String() string {
	out := "Fileinfos:\n"
	for _, e := range m {
		out += "  " + e.Name() + "\n"
	}
	return out
}

func ReaddirAll(t *testing.T, fs afero.Fs) {
	tmpDir := mustTempDir(fs)
	defer fs.RemoveAll(tmpDir)
	testSubDir := setupTestFiles(t, fs, tmpDir)
	tDir := filepath.Dir(testSubDir)

	root, err := fs.Open(tDir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	rootInfo, err := root.Readdir(-1)
	if err != nil {
		t.Fatal(err)
	}
	var namesRoot = []string{}
	for _, e := range rootInfo {
		namesRoot = append(namesRoot, e.Name())
	}

	sub, err := fs.Open(testSubDir)
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Close()

	subInfo, err := sub.Readdir(-1)
	if err != nil {
		t.Fatal(err)
	}
	var namesSub = []string{}
	for _, e := range subInfo {
		namesSub = append(namesSub, e.Name())
	}

	findNames(fs, t, tDir, testSubDir, namesRoot, namesSub)
}

func StatDirectory(t *testing.T, fs afero.Fs) {
	tmpDir := mustTempDir(fs)
	defer fs.RemoveAll(tmpDir)
	dirName := setupTestFiles(t, fs, tmpDir)

	for _, validStatInput := range []string{
		dirName,
		dirName + "/",
		dirName + "//",
		"/" + dirName,
	} {
		dirStat, err := fs.Stat(validStatInput)
		if err != nil {
			t.Fatalf("could not stat %s: %s", validStatInput, err)
		}
		if expected := filepath.Base(dirName); dirStat.Name() != expected {
			t.Fatalf("valid stat input: %s got Name %s, expected %s",
				validStatInput, dirStat.Name(), expected)
		}
		if !dirStat.IsDir() {
			t.Fatalf("valid stat input: %s got IsDir false for directory",
				validStatInput)
		}
	}

	for _, invalidStatInput := range []string{
		dirName + "doesntexist",
		dirName + "doesntexist/",
		dirName + "/doesntexist",
		dirName + "/doesntexist/",
	} {
		if _, err := fs.Stat(invalidStatInput); err == nil {
			t.Fatalf("invalid stat input: %s no error returned for non-existent directory",
				invalidStatInput)
		} else if !os.IsNotExist(err) {
			t.Fatalf("invalid stat input: %s error returned from Stat does not pass `os.IsNotExist` test: %s",
				invalidStatInput, err)
		}
	}
}

func StatFile(t *testing.T, fs afero.Fs) {
	f := mustTempFile(fs)
	defer f.Close()
	defer fs.Remove(f.Name())
	fileName := f.Name()
	for _, validStatInput := range []string{
		fileName,
		"/" + fileName,
	} {
		fileStat, err := fs.Stat(validStatInput)
		if err != nil {
			t.Errorf("could not stat %s: %s", validStatInput, err)
			continue
		}
		if expected := filepath.Base(fileName); fileStat.Name() != expected {
			t.Errorf("valid stat input: %s got Name %s, expected %s",
				validStatInput, fileStat.Name(), expected)
		}
		if fileStat.IsDir() {
			t.Errorf("valid stat input: %s got IsDir true for file",
				validStatInput)
		}
	}

	for _, invalidStatInput := range []string{
		fileName + "doesntexist",
		fileName + "doesntexist/",
		fileName + "/",
	} {
		if _, err := fs.Stat(invalidStatInput); err == nil {
			t.Errorf("invalid stat input: %s no error returned for non-existent file",
				invalidStatInput)
		} else if !os.IsNotExist(err) {
			t.Fatalf("invalid stat input: %s error returned from Stat does not pass `os.IsNotExist` test: %s",
				invalidStatInput, err)
		}
	}
}

func findNames(fs afero.Fs, t *testing.T, tDir, testSubDir string, root, sub []string) {
	var foundRoot bool
	for _, e := range root {
		f, err := fs.Open(filepath.Join(tDir, e))
		if err != nil {
			t.Error("Open", filepath.Join(tDir, e), ":", err)
		}
		defer f.Close()

		if equal(e, "we") {
			foundRoot = true
		}
	}
	if !foundRoot {
		t.Logf("Names root: %v", root)
		t.Logf("Names sub: %v", sub)
		t.Error("Didn't find subdirectory we")
	}

	var found1, found2 bool
	for _, e := range sub {
		f, err := fs.Open(filepath.Join(testSubDir, e))
		if err != nil {
			t.Error("Open", filepath.Join(testSubDir, e), ":", err)
		}
		defer f.Close()

		if equal(e, "testfile1") {
			found1 = true
		}
		if equal(e, "testfile2") {
			found2 = true
		}
	}

	if !found1 {
		t.Logf("Names root: %v", root)
		t.Logf("Names sub: %v", sub)
		t.Error("Didn't find testfile1")
	}
	if !found2 {
		t.Logf("Names root: %v", root)
		t.Logf("Names sub: %v", sub)
		t.Error("Didn't find testfile2")
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

func checkSize(t *testing.T, f afero.File, size int64) {
	dir, err := f.Stat()
	if err != nil {
		t.Fatalf("Stat %q (looking for size %d): %s", f.Name(), size, err)
	}
	if dir.Size() != size {
		t.Errorf("Stat %q: size %d want %d", f.Name(), dir.Size(), size)
	}
}
