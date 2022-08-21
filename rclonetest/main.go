package main

import (
	"fmt"
	"os"

	"github.com/spf13/afero/rclonefs"
)

func main() {
	RFS, _ := rclonefs.CreateRCFS("pcloud_mv1:/cfg/json")

	file, err := RFS.OpenFile("../../obj1.json", os.O_RDWR | os.O_TRUNC, 0666)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	file.WriteString("{\"game\":\"Max Payne\"}")
	file.Close()
}
