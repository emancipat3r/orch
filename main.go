package main

import (
	"os"
	"os/user"

	"github.com/charmbracelet/keygen"
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

	var pathConfig, pathSSH, privateKeyPath, publicKeyPath string

	pathConfig = user.HomeDir + "/.config/vps/config/"
	pathSSH = user.HomeDir + "/.config/vps/.ssh/"
	pathSecrets = user.HomeDir + "/.config/vps/secrets/"
	privateKeyPath = user.HomeDir + "/.config/vps/.ssh/id_ed25519"
	publicKeyPath = user.HomeDir + "/.config/vps/.ssh/id_ed25519.pub"

	// Check if config directory exists
	if utils.DirExists(pathConfig) {
		logger.Info("Provider configuration directory exists: " + logger.Highlight(pathConfig) + ". Moving along...")
	} else {
		logger.Warn("Provider configuration directory doesn't exist. Creating - " + logger.Highlight(pathConfig))
		err := utils.MakeDirectory(pathConfig)
		if err != nil {
			logger.Error("Failed to create directory: " + err.Error())
			return
		}

		// Check if directory was created successfully
		if utils.DirExists(pathConfig) {
			logger.Success("Directory created: " + logger.Highlight(pathConfig))
		} else {
			logger.Error("Failed to create directory. Exiting...")
			return
		}
	}

	// Check if SSH directory exists
	if utils.DirExists(pathSSH) {
		logger.Info("Provider SSH directory exists: " + logger.Highlight(pathSSH) + ". Moving along...")
	} else {
		logger.Warn("Provider SSH directory doesn't exist. Creating - " + logger.Highlight(pathSSH))
		err := utils.MakeDirectory(pathSSH)
		if err != nil {
			logger.Error("Failed to create directory: " + err.Error())
			return
		}

		// Check if directory was created successfully
		if utils.DirExists(pathSSH) {
			logger.Success("Directory created: " + logger.Highlight(pathSSH))
		} else {
			logger.Error("Failed to create directory. Exiting...")
			return
		}
	}

	// Check for credentials file
	configFile := pathConfig + "configuration.toml"
	if utils.FileExists(configFile) {
		logger.Info("Provider configuration file exists: " + logger.Highlight(configFile))
	} else {
		logger.Warn("Provider configuration file missing. Exiting...")
		return
	}

	// Check for SSH keys
	if utils.FileExists(privateKeyPath) && utils.FileExists(publicKeyPath) {
		logger.Info("SSH private key exists: " + logger.Highlight(privateKeyPath))
		logger.Info("SSH public key exists: " + logger.Highlight(publicKeyPath))
	} else {
		logger.Warn("SSH keypair missing. Creating...")

		// Generate random password
		pass, err = utils.GenerateRandomPassword()
		if err != nil {
			logger.Error("Failed to generate random password for SSH keypair: " + err.Error())
			return
		}

		// Write random password to disk
		_ = os.WriteFile(pathSecrets+"sshkeysecret.txt", []byte(password+"\n"), 0600)

		// Create keypair
		k, err := keygen.New(pathSSH, keygen.WithKeyType(keygen.Ed25519), keygen.WithPassphrase(pass))
		if err != nil {
			logger.Error("Failed to create SSH keypair: " + err.Error())
			return err
		}
		
		// Check if privateKeyPath and publicKeyPath are true
		if {
			logger.Success(": " + logger.Highlight(pathSSH))
		} else {

		}

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
	// [x] Get List of flavors (e.g. Linux distributions)
	// [x] User choice flavor
	// [ ] Get list of resource options for selected flavor
	switch provider {
	case "Linode":
		// Fetch account balance
		accountBalance, _ := providers.GetLinodesBalance(providerKey)
		logger.Info("Linode account balance: " + logger.Highlight(accountBalance))

		// Fetch regions
		regions, err := providers.GetLinodeRegions()
		if err != nil {
			logger.Error("Failed to hit endpoint: " + err.Error())
			return
		}

		// Add requested / parsed JSON to regionOptions slice
		var regionOptions []string
		for _, region := range regions {
			if region.Status == "ok" {
				regionOptions = append(regionOptions, region.ID+" - "+region.Label)
			}
		}

		// Ask user region
		selectedRegion := ui.Select("Select your region:", regionOptions)
		logger.Info("You selected region: " + logger.Highlight(selectedRegion))

		// Fetch image types
		images, err := providers.GetLinodeImages()
		if err != nil {
			logger.Error("Failed to hit endpoint: " + err.Error())
			return
		}

		// Add requested/parsed JSON to imageOptions slice
		var imageOptions []string
		for _, image := range images {
			imageOptions = append(imageOptions, image.Label)
		}

		// Ask user image
		selectedImage := ui.Select("Select your image:", imageOptions)
		logger.Info("Your selected image: " + logger.Highlight(selectedImage))

		// Earlier in proc need to check for / create a SSH key

		// Fetch resources

		// Ask user resources

		// Send create linode request

		// Polling for when the Linode is up
		// and available for work
		//	- Wait 10s then curl
		//
		//	---------------------------------------------------
		//
		// Apparently you can curl to see the status of an endpoint
		// (e.g. provisioning or running)
		//
		//	>    curl https://api.linode.com/v4/linode/instances \
		//	>	-H "Authorization: Bearer $TOKEN" \
		//	>	-H 'X-Filter: { "id": "linode_ID" }'

		//

	case "DigitalOcean":
		logger.Info("Work in progress...")
	case "Vultr":
		logger.Info("Work in progress...")
	default:
		logger.Warn("No provider was selected. Exiting...")
	}

}
