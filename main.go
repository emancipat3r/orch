package main

import (
	"fmt"

	"github.com/emancipat3r/vps3/utils"
)

func main() {
	/*
		utils.Info("My Logger")
		utils.Debug("Debug")
		utils.Error("Error")
		utils.Warn("Warn")
	*/

	path := utils.Prompt("What is your path? > ")

	fmt.Println(path)

	if utils.DirExists(path) {
		fmt.Println("Directory exists")
	} else {
		fmt.Println("Directory doesn't exist")
	}
}
