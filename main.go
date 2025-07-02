package main

import (
	"os/user"

	"github.com/emancipat3r/vps3/logger"
	"github.com/emancipat3r/vps3/providers"
	"github.com/emancipat3r/vps3/ui"
	"github.com/emancipat3r/vps3/utils"
)

func main() {
	user, err := user.Current()
	if err != nil {
		logger.Error("Failed to get current user: " + err.Error())
		return
	}

	var path string

	if user.Username == "root" {
		path = "/root/.config/vps/config/"
	} else {
		path = user.HomeDir + "/.config/vps/config/"
	}

	// Check if directory exists
	if utils.DirExists(path) {
		logger.Info("Provider configuration directory exists: " + logger.Highlight(path) + ". Moving along...")
	} else {
		logger.Warn("Provider configuration directory doesn't exist. Creating - " + logger.Highlight(path))
		err := utils.MakeDirectory(path)
		if err != nil {
			logger.Error("Failed to create directory: " + err.Error())
			return
		}

		// Check if directory was created successfully
		if utils.DirExists(path) {
			logger.Success("Directory created: " + logger.Highlight(path))
		} else {
			logger.Error("Failed to create directory. Exiting...")
			return
		}
	}

	// Check for credentials file
	configFile := path + "configuration.toml"
	if utils.FileExists(configFile) {
		logger.Info("Provider configuration file exists: " + logger.Highlight(configFile))
	} else {
		logger.Warn("Provider configuration file missing. Exiting...")
		return
	}

	// Ask for provider
	provider := ui.ChoiceProvider()

	logger.Info("You selected: " + logger.Highlight(provider))

	// Parse provider credentials from configuration file
	providerKey := providers.GetLinodeAPIKey(configFile, provider)

	// Print provider token/api
	logger.Info("Provider Key: " + logger.Highlight(providerKey))

	// [x] Get list of regions
	// [x] User choice of region
	// [ ] Get List of flavors (e.g. Linux distributions)
	// [ ] User choice on
	// [ ] Get list of resource options for selected flavor
	switch provider {
	case "Linode":
		accountBalance, _ := providers.GetLinodesBalance(providerKey)
		logger.Info("Linode account balance: " + logger.Highlight(accountBalance))

		regions, err := providers.GetLinodeRegions()
		if err != nil {
			logger.Error("Failed to hit endpoint: " + err.Error())
			return
		}

		var regionOptions []string
		for _, region := range regions {
			if region.Status == "ok" {
				regionOptions = append(regionOptions, region.ID+" - "+region.Label)
			}
		}

		selectedRegion := ui.Select("Select your region:", regionOptions)
		logger.Info("You selected region: " + logger.Highlight(selectedRegion))

		images, err := providers.GetLinodeImages()
		if err != nil {
			logger.Error("Failed to hit endpoint: " + err.Error())
			return
		}

		var imageOptions []string
		for _, image := range images {
			imageOptions = append(imageOptions, image.Label)
		}

		selectedImage := ui.Select("Select your image:", imageOptions)
		logger.Info("Your selected image: " + logger.Highlight(selectedImage))

	case "DigitalOcean":
		logger.Info("Work in progress...")
	case "Vultr":
		logger.Info("Work in progress...")
	default:
		logger.Warn("No provider was selected. Exiting...")
	}

}
