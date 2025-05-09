package utils

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"os"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func MakeTempDir() {
	tmpdir, err := os.MkdirTemp("", "tmp.*")
	check(err)
	fmt.Println("Temp dir created -", tmpdir)
}
