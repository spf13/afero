package ossfs

import "os"

func test() {
	f, _ := os.Open("test")
	f.Readdir()
}
