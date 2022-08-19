package main

import (
	"fmt"
	"os"

	"github.com/spf13/afero"
	"github.com/spf13/afero/rclonefs"
)

func main() {
	RFS, _ := rclonefs.CreateRCFS("godrive1:")

	data, err := afero.ReadFile(RFS, "mock.json")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", string(data))
}
