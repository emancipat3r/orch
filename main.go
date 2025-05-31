package main

import (
	"github.com/emancipat3r/vps3/logger"
	"github.com/emancipat3r/vps3/ui"
	"github.com/emancipat3r/vps3/utils"
)

func main() {
	path := "./creds"
	logger.Info(path)

	// Check if directory exists
	if utils.DirExists(path) {
		logger.Info("Creds directory exists")
	} else {
		logger.Warn("Creds directory doesn't exist. Creating...")
		err := utils.MakeDirectory(path)
		if err != nil {
			logger.Error("Failed to create directory: " + err.Error())
			return
		}

		// Check if directory was created successfully
		if utils.DirExists(path) {
			logger.Success("Directory created - " + path)
		} else {
			logger.Error("Failed to create directory. Exiting...")
			return
		}
	}

	// Check for credentials file
	if utils.FileExists(path + "/credentials.json") {
		logger.Info("Credentials file exists")
	} else {
		logger.Warn("Credentials file missing")
		shouldLoad := ui.Confirm("Do you want to provide a path to an existing credentials file?")
		if shouldLoad {
			newPath := ui.Input("Enter path to credentials file:")
			if utils.FileExists(newPath) {
				// Copy or move file
				utils.CopyFile(newPath, path+"/credentials.json")
				logger.Success("Credentials loaded.")
			} else {
				logger.Error("Provided file does not exist.")
			}
		} else {
			logger.Warn("No credentials loaded. Exiting...")
			return
		}
	}
}
