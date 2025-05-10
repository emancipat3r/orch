package main

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/emancipat3r/vps3/utils"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	utils.MakeTempDir()
}
