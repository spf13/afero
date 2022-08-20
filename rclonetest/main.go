package main

import (
	"fmt"
	"os"

//	"github.com/spf13/afero"
	"github.com/spf13/afero/rclonefs"
)

func main() {
	RFS, _ := rclonefs.CreateRCFS("pcloud_mv1:/cfg")

	name := "ycfg"

	err := RFS.RemoveAll(name)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
