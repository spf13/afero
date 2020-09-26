package swift

import (
	"bytes"
	"fmt"
	"github.com/ncw/swift"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"testing"
	"time"
)

type testFile struct {
	name string
	typ  string
}

func (s *Fs) prepareTestContainer() {
	// Get all existing objects to clear the container
	objects, err := s.Connection.ObjectsAll(s.containerName, nil)
	if err != nil {
		log.Fatal(err)
	}
	info, err := s.Connection.QueryInfo()
	if err != nil {
		log.Fatal(err)
	}

	var objectnames []string
	for _, o := range objects {
		if info.SupportsBulkDelete() {
			objectnames = append(objectnames, o.Name)
		} else {
			err = s.Connection.ObjectDelete(s.containerName, o.Name)
			if err != nil {
				log.Fatal(err)
			}
			log.Println("Deleted object", o.Name)
		}
	}
	// Bulk delete all in one go if the server supports it
	if info.SupportsBulkDelete() {
		_, err = s.Connection.BulkDelete(s.containerName, objectnames)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Deleted", objectnames)
	}
	// Create some files we're expecting
	for _, newObject := range []testFile{
		{"existingTestfile", "application/octet-stream"},
		{"fileToDelete", "application/octet-stream"},
		{"testfolder/", "application/directory"},
		{"testfolder/fileintestfolder1", "application/octet-stream"},
		{"testfolder/fileintestfolder2", "application/octet-stream"},
		{"testfolder/subfolder/", "application/directory"},
		{"testfolder/subfolder/fileinsubtestfolder1", "application/octet-stream"},
		{"testfolder/subfolder/fileinsubtestfolder2", "application/octet-stream"},
	} {
		_, err := s.Connection.ObjectPut(s.containerName, newObject.name, bytes.NewReader([]byte{}), true, "", newObject.typ, swift.Headers{})
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Created object", newObject)
	}
}

func (s *Fs) checkExpectedContent(files []testFile) (notmatching []testFile) {
	objects, err := s.Connection.ObjectsAll(s.containerName, nil)
	if err != nil {
		log.Fatal(err)
	}

	// We use this map to keep track of files we already passed so we don't count them twice
	filesfound := make(map[string]testFile)
	// We use this map to keep track of all files we wanted, but didn't exist.
	// When all files we want to exist exist, this map should be empty at the end of this function.
	filesnotpassed := make(map[string]testFile)
	for _, o := range objects {
		for _, ob := range files {
			filesnotpassed[ob.name] = ob
			if o.Name == ob.name && o.ContentType == ob.typ {
				filesfound[o.Name] = ob
				delete(filesnotpassed, o.Name)
				continue
			}
			if _, ok := filesfound[o.Name]; ok {
				continue
			}
		}
	}

	for _, o := range objects {
		if _, ok := filesfound[o.Name]; !ok {
			delete(filesnotpassed, o.Name)
			// Notmatching will contain all files which exist, but we didn't want to exist.
			notmatching = append(notmatching, testFile{o.Name, o.ContentType})
		}
	}

	// Append all files we did not found
	for _, f := range filesnotpassed {
		if _, ok := filesfound[f.name]; !ok {
			notmatching = append(notmatching, f)
		}
	}

	return
}

// Helper method to get a testinstance
func createSwiftfsTestInstance() (fs *Fs, err error) {
	// Maybe we can cache this somehow? (aka not creating a new connection for every test function)
	// Also we need a good way to set up fixtures before each test.
	fs, err = NewSwiftFs(&swift.Connection{
		UserName: os.Getenv("SWIFT_APIUSER"),
		ApiKey:   os.Getenv("SWIFT_APIKEY"),
		AuthUrl:  os.Getenv("SWIFT_AUTHURL"),
		Domain:   os.Getenv("SWIFT_DOMAIN"),
	}, "testcontainer")
	if err != nil {
		return
	}
	fs.prepareTestContainer()
	return
}

var testStringToWrite = "lorem ipsum"

func TestFs_Name(t *testing.T) {
	swiftfs, err := createSwiftfsTestInstance()
	assert.NoError(t, err)

	assert.Equal(t, "swiftfs", swiftfs.Name())
}

func TestFsCreate(t *testing.T) {
	swiftfs, err := createSwiftfsTestInstance()
	assert.NoError(t, err)

	file, err := swiftfs.Create("testfile")
	assert.NoError(t, err)

	// Test writing to a created file
	written, err := file.WriteString(testStringToWrite)
	assert.NoError(t, err)
	assert.Equal(t, len(testStringToWrite), written)

	err = file.Close()
	assert.NoError(t, err)
}

func TestFs_Open(t *testing.T) {
	swiftfs, err := createSwiftfsTestInstance()
	assert.NoError(t, err)

	// Try opening a file which does not exist
	_, err = swiftfs.Open("notExistingTestfile")
	assert.Error(t, err)

	// Open the real file
	file, err := swiftfs.Open("existingTestfile") // This file needs to exist previously
	assert.NoError(t, err)

	// Test writing to an open file
	written, err := file.WriteString(testStringToWrite)
	assert.NoError(t, err)
	assert.Equal(t, len(testStringToWrite), written)

	err = file.Close()
	assert.NoError(t, err)
}

func TestFs_Mkdir(t *testing.T) {
	swiftfs, err := createSwiftfsTestInstance()
	assert.NoError(t, err)

	err = swiftfs.Mkdir("testdir", os.ModePerm)
	assert.NoError(t, err)
}

// MkdirAll should do exactly the same
func TestFs_MkdirAll(t *testing.T) {
	swiftfs, err := createSwiftfsTestInstance()
	assert.NoError(t, err)

	err = swiftfs.MkdirAll("testdir", os.ModePerm)
	assert.NoError(t, err)
}

func TestFs_Remove(t *testing.T) {
	swiftfs, err := createSwiftfsTestInstance()
	assert.NoError(t, err)

	err = swiftfs.Remove("fileToDelete")
	assert.NoError(t, err)

	// Testing removing a second time should return an error as that file does not exist now
	err = swiftfs.Remove("fileToDelete")
	assert.Error(t, err)
}

func TestFs_RemoveAll(t *testing.T) {
	swiftfs, err := createSwiftfsTestInstance()
	assert.NoError(t, err)

	err = swiftfs.RemoveAll("testfolder")
	assert.NoError(t, err)
	notmatching := swiftfs.checkExpectedContent([]testFile{
		{"existingTestfile", "application/octet-stream"},
		{"fileToDelete", "application/octet-stream"},
	})
	assert.Len(t, notmatching, 0)

	// Deleting something with removeAll should return nil if it does not exist
	err = swiftfs.RemoveAll("testfolder")
	assert.NoError(t, err)
}

func TestFs_Rename(t *testing.T) {
	swiftfs, err := createSwiftfsTestInstance()
	assert.NoError(t, err)

	err = swiftfs.Rename("existingTestfile", "otherExistingTestfile")
	assert.NoError(t, err)
	notmatching := swiftfs.checkExpectedContent([]testFile{
		{"otherExistingTestfile", "application/octet-stream"},
		{"fileToDelete", "application/octet-stream"},
		{"testfolder/", "application/directory"},
		{"testfolder/fileintestfolder1", "application/octet-stream"},
		{"testfolder/fileintestfolder2", "application/octet-stream"},
		{"testfolder/subfolder/", "application/directory"},
		{"testfolder/subfolder/fileinsubtestfolder1", "application/octet-stream"},
		{"testfolder/subfolder/fileinsubtestfolder2", "application/octet-stream"},
	})
	assert.Len(t, notmatching, 0)

	// Moving a nonexisting file should not work
	err = swiftfs.Rename("existingTestfile", "otherExistingTestfile")
	assert.Error(t, err)
}

func TestFs_Stat(t *testing.T) {
	swiftfs, err := createSwiftfsTestInstance()
	assert.NoError(t, err)

	finfo, err := swiftfs.Stat("existingTestfile")
	assert.NoError(t, err)
	assert.Equal(t, "existingTestfile", finfo.Name())
	assert.Equal(t, int64(0), finfo.Size())
	assert.Equal(t, os.ModePerm, finfo.Mode())
	assert.False(t, finfo.IsDir())
	assert.Equal(t, finfo, finfo.Sys())
}

func TestFs_Chmod(t *testing.T) {
	swiftfs, err := createSwiftfsTestInstance()
	assert.NoError(t, err)

	err = swiftfs.Chmod("existingTestfile", os.ModePerm)
	assert.NoError(t, err)
}

func TestFs_Chtimes(t *testing.T) {
	swiftfs, err := createSwiftfsTestInstance()
	assert.NoError(t, err)

	filename := "existingTestfile"
	newtime := time.Date(2019, time.January, 31, 14, 0, 0, 0, time.UTC)
	err = swiftfs.Chtimes(filename, newtime, newtime)
	assert.NoError(t, err)

	// See the function on why this check is diabled.
	/*
		file, err := swiftfs.Open(filename)
		assert.NoError(t, err)
		finfo, err := file.Stat()
		assert.NoError(t, err)
		assert.True(t, newtime.Equal(finfo.ModTime()))
	*/
}

func TestReaddir(t *testing.T) {
	swiftfs, err := createSwiftfsTestInstance()
	assert.NoError(t, err)

	root, err := swiftfs.Open("testfolder/")
	assert.NoError(t, err)

	folders, err := root.Readdir(10) // Random number
	assert.NoError(t, err)

	numfolders := 2
	for _, f := range folders {
		if f == nil {
			continue
		}
		if f.IsDir() {
			numfolders--
			continue
		}

		fmt.Printf("%s is not a folder", f.Name())
		t.FailNow()
	}

	assert.Equal(t, 0, numfolders)
}
