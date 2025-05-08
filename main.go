package main

import (
	"fmt"
	"os"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func makeTempDir() {
	tmpdir, err := os.MkdirTemp("", "tmp.*")
	check(err)
	fmt.Println("Temp dir created -", tmpdir)
}

func main() {
	makeTempDir()
}
