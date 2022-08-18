package main

import (
	"github.com/spf13/afero"
	"github.com/spf13/afero/rclonefs"
)

func main() {
	newrc := rclonefs.CreateRCFS("godrive_mv1:")
}
