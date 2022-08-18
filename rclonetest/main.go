package main

import (
	"fmt"

	"github.com/spf13/afero/rclonefs"
)

func main() {
	newrc, _ := rclonefs.CreateRCFS("godrive1:")

	fmt.Println(newrc)
}
