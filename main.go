package main

import (
	"os/user"

	"github.com/emancipat3r/vps3/logger"
	"github.com/emancipat3r/vps3/ui"
	"github.com/emancipat3r/vps3/utils"
)

func main() {
	user, err := user.Current()
	if err != nil {
		logger.Error("Failed to get current user: " + err.Error())
		return
	}

	path := ""

	if user.Username == "root" {
		path = "/root/.config/vps/config/"
	} else {
		path = user.HomeDir + "/.config/vps/config/"
	}

	// Check if directory exists
	if utils.DirExists(path) {
		logger.Info("Provider configuration directory exists: " + path + ". Moving along...")
	} else {
		logger.Warn("Provider configuration directory doesn't exist. Creating - " + path)
		err := utils.MakeDirectory(path)
		if err != nil {
			logger.Error("Failed to create directory: " + err.Error())
			return
		}

		// Check if directory was created successfully
		if utils.DirExists(path) {
			logger.Success("Directory created: " + path)
		} else {
			logger.Error("Failed to create directory. Exiting...")
			return
		}
	}

	// Check for credentials file
	configFile := path + "configuration.toml"
	if utils.FileExists(configFile) {
		logger.Info("Provider configuration file exists: " + configFile)
	} else {
		logger.Warn("Provider configuration file missing. Exiting...")
		return
	}

	// Ask for provider
	provider := ui.ChoiceProvider()

	// Parse provider credentials from configuration file
	providerKey := utils.GetLinodeAPIKey(configFile, provider)

	logger.Info("Provider Key: " + providerKey)

	utils.GetLinodeAccount(providerKey)
}
