package main

import (
	"fmt"
	"os"

	"github.com/spf13/afero/rclonefs"
)

func main() {
	RFS, _ := rclonefs.CreateRCFS("pcloud_mv1:/cfg/json")


	f, e := RFS.Create("../obj3.json")
	if e != nil {
		fmt.Printf("Error: %v\n", e)
		os.Exit(1)
	}

	defer f.Close()
}
