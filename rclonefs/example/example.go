// Example and test
package main

import (
//	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"log"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/afero/rclonefs"
)

func main() {
	rawpath, err := os.ReadFile("cloud.txt")
	if err != nil {
		log.Fatalf("Error in ReadFile: %v\n", err)
	}

	path := strings.TrimSpace(string(rawpath))

	RFS, err := rclonefs.CreateRCFS(path, "")
	if err != nil {
		log.Fatalf("Error in CreateRCFS: %v\n", err)
	}

	// Test examples start.
	// File path beginning from / is the absolute path.
	// Otherwise it is the relative path.
	fmt.Printf("Checking...\n")

	// Create() test
	file1, err := RFS.Create("/cfg/json/object.json") // absolute path
	if err != nil {
		log.Fatal("Create() test FAILED\n")
	}
	file1.Close()
	ok, err := afero.Exists(RFS, "object.json") // relative path
	if err != nil {
		log.Fatalf("Error in Exists(): %v\n", err)
	}
	if ok {
		fmt.Printf("Create() test passed.\n")
	}

	// Mkdir() test
	err = RFS.Mkdir("../yml", 0750) // absolute path: /cfg/yml
	if err != nil {
		log.Fatal("Mkdir() test FAILED\n")
	}
	ok, err = afero.DirExists(RFS, "../yml")
	if err != nil {
		log.Fatalf("Error in DirExists(): %v\n", err)
	}
	if ok {
		fmt.Printf("Mkdir() test passed.\n")
	}

	// MkdirAll() test
	err = RFS.MkdirAll("../../data/passwords/sites", 0750) // absolute path: /data/passwords/sites
	if err != nil {
		log.Fatal("MkdirAll() test FAILED\n")
	}
	ok, err = afero.DirExists(RFS, "/data/passwords/sites")
	if err != nil {
		log.Fatalf("Error in DirExists(): %v\n", err)
	}
	if ok {
		fmt.Printf("MkdirAll() test passed.\n")
	}

	// afero.WriteFile() test - the function is based on OpenFile()
	// copying object.json to remote storage
	data, err := os.ReadFile("object.json")
	if err != nil {
		log.Fatalf("Error in ReadFile(): %v\n", err)
	}
	afero.WriteFile(RFS, "object.json", data, 0644)

	// Open() test
	file1, err = RFS.Open("object.json")
	if err != nil {
		log.Fatal("Open() test FAILED\n")
	}

	// Stat() test
	if fi, err := file1.Stat(); err == nil {
		if size := fi.Size(); size == 129 {
			fmt.Printf("Stat() test passed.\n")
		}
	}
	file1.Close()

	// afero.ReadFile() test - the function is based on Open()
	data, err = afero.ReadFile(RFS, "object.json")
	if err != nil {
		log.Fatalf("Error in afero.ReadFile(): %v\n", err)
	}

	mp := make(map[string]interface{})
	err = json.Unmarshal(data, &mp)
	if err != nil {
		log.Fatalf("Error in json.Unmarshal(): %v\n", err)
	}

	var test2 int = int(mp["test2"].(float64))
	if test2 != 7 {
		log.Fatal("afero.ReadFile() / Open() test FAILED")
	}

	fmt.Printf("Open(): all tests passed.\n")
	fmt.Printf("OpenFile() test passed.\n")

	// GetTempDir example
	// Relative paths like "../cfg" do not work.
	afero.GetTempDir(RFS, "cfg")

	// SafeWriteReader example
	srcfile, err := os.OpenFile("call.json", os.O_RDWR, 0644)
	if err != nil {
		log.Fatalf("Error in os.OpenFile(): %v\n", err)
	}

	err = afero.SafeWriteReader(RFS, "../../data/passwords/sites/github/call.json", srcfile)
	if err != nil {
		log.Fatalf("SafeWriteReader() test FAILED")
	}
	srcfile.Close()

	ok, err = afero.Exists(RFS, "/data/passwords/sites/github/call.json")
	if err != nil {
		log.Fatalf("Error in afero.Exists(): %v\n", err)
	}
	if ok {
		fmt.Printf("SafeWriteReader() / MkdirAll() test passed.\n")
	}

	// WriteReader example
	srcfile, err = os.OpenFile("employee.yml", os.O_RDWR, 0644)
	if err != nil {
		log.Fatalf("Error in os.OpenFile(): %v\n", err)
	}

	err = afero.WriteReader(RFS, "/data/passwords/employee.yml", srcfile)
	if err != nil {
		log.Fatalf("WriteReader() test FAILED")
	}
	srcfile.Close()

	ok, err = afero.Exists(RFS, "/data/passwords/employee.yml")
	if err != nil {
		log.Fatalf("Error in afero.Exists(): %v\n", err)
	}
	if ok {
		fmt.Printf("WriteReader() / OpenFile() test passed.\n")
	}

	// TempDir() example
	tdir, err := afero.TempDir(RFS, "../../cfg", "tmp")
	if err != nil {
		log.Fatalf("Error in afero.TempDir(): %v\n", err)
	}

	ok, err = afero.DirExists(RFS, tdir)
	if err != nil {
		log.Fatalf("Error in DirExists(): %v\n", err)
	}
	if ok {
		fmt.Printf("TempDir() test passed.\n")
	}

	// Remove() test
	err = RFS.Remove("/data/passwords/employee.yml")
	if err != nil {
		log.Fatal("Remove() test FAILED\n")
	}

	ok, err = afero.Exists(RFS, "/data/passwords/employee.yml")
	if err != nil {
		log.Fatal("Error in afero.Exists(): %v\n", err)
	}
	if !ok {
		fmt.Printf("Remove() test passed.\n")
	}

	// RemoveAll() test
	err = RFS.RemoveAll("../../data/passwords") // relative path
	if err != nil {
		log.Fatal("RemoveAll() test FAILED\n")
	}

	ok, err = afero.DirExists(RFS, "/data/passwords") // absolute path
	if err != nil {
		log.Fatal("Error in afero.DirExists(): %v\n", err)
	}
	if !ok {
		fmt.Printf("RemoveAll() test passed.\n")
	}

	// Rename() test
	err = RFS.Rename("object.json", "object2.json") // relative paths
	if err != nil {
		log.Fatal("Rename() test FAILED\n")
	}

	ok, err = afero.Exists(RFS, "/cfg/json/object2.json") // absolute path
	if err != nil {
		log.Fatal("Error in afero.Exists(): %v\n")
	}
	if ok {
		fmt.Printf("Rename() test passed.\n")
	}

	fmt.Printf("ALL TESTS PASSED\n")
}
