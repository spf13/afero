package main

import (
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

	// Create() test
	file1, err := RFS.Create("/cfg/json/file1.json") // absolute path
	if err != nil {
		log.Fatal("Create() test FAILED\n")
	}
	file1.Close()
	ok, err := afero.Exists(RFS, "file1.json") // relative path
	if err != nil {
		log.Fatal("Error in Exists(): %v\n")
	}
	if ok {
		fmt.Printf("Create() test passed.\n")
	}
}
