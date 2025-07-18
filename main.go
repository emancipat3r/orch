package main

import (
	"os"
	"os/user"
	"strings"

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

	var pathConfig, pathSSH, pathSecrets, privateKeyPath, publicKeyPath string

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

		var password string
		// Generate random password
		password, err = utils.GenerateRandomPassword(30)
		if err != nil {
			logger.Error("Failed to generate random password for SSH keypair: " + err.Error())
			return
		}

		// Write random password to disk
		_ = os.WriteFile(pathSecrets+"sshkeysecret.txt", []byte(password+"\n"), 0600)

		// Create keypair
		_, err = keygen.New(privateKeyPath, keygen.WithKeyType(keygen.Ed25519), keygen.WithPassphrase(password), keygen.WithWrite())
		if err != nil {
			logger.Error("Failed to create SSH keypair: " + err.Error())
			return
		}

		// Check if privateKeyPath and publicKeyPath are true
		if utils.FileExists(privateKeyPath) && utils.FileExists(publicKeyPath) {
			logger.Info("SSH private key exists: " + logger.Highlight(privateKeyPath))
			logger.Info("SSH public key exists: " + logger.Highlight(publicKeyPath))
		} else {
			logger.Error("Still failed to create SSH keypair (files still missing).")
			return
		}
	}

	// Ask for provider
	provider := ui.ChoiceProvider()

	logger.Info("You selected: " + logger.Highlight(provider))

	// Parse provider credentials from configuration file
	providerKey := providers.GetLinodeAPIKey(configFile, provider)

	switch provider {
	case "Linode":
		// Fetch account balance
		accountBalance, err := providers.GetLinodesBalance(providerKey)
		if err != nil {
			os.Exit(1)
		}
		logger.Info("Linode account balance: " + logger.Highlight("$"+accountBalance))

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

		selectedRegionSplit := strings.Split(selectedRegion, " ")

		// Fetch image types
		images, err := providers.GetLinodeImages()
		if err != nil {
			logger.Error("Failed to hit endpoint: " + err.Error())
			return
		}

		// Add requested/parsed JSON to imageOptions slice
		var imageOptions []string
		for _, image := range images {
			imageOptions = append(imageOptions, image.ID+" - "+image.Label)
		}

		// Ask user image
		selectedImage := ui.Select("Select your image:", imageOptions)
		logger.Info("Your selected image: " + logger.Highlight(selectedImage))

		selectedImageSplit := strings.Split(selectedImage, " ")

		// Fetch resources
		resources, err := providers.GetLinodeResources()
		if err != nil {
			logger.Error("Failed to hit endpoint: " + err.Error())
			return
		}

		// Add requested / parsed JSON to resourceOptions slice
		var resourceOptions []string
		for _, resource := range resources {
			resourceOptions = append(resourceOptions, resource.ID+" - "+resource.Label)
		}

		// Ask user region
		selectedResource := ui.Select("Select your resourcing:", resourceOptions)
		logger.Info("You selected type: " + logger.Highlight(selectedResource))

		selectedResourceSplit := strings.Split(selectedResource, " ")

		var rootPassword string
		// Generate random password
		rootPassword, err = utils.GenerateRandomPassword(30)
		if err != nil {
			logger.Error("Failed to generate root password for Linode: " + err.Error())
			return
		}

		// Create Linode
		logger.Info("Creating Linode...")
		createResponse, err := providers.CreateLinode(providerKey, publicKeyPath, selectedImageSplit[0], selectedRegionSplit[0], selectedResourceSplit[0], rootPassword)
		if err != nil {
			logger.Error("Failed to create Linode: " + err.Error())
		}
		logger.Debug(logger.Highlight("Response:\n") + createResponse)

		// Write json output to instance file
		// Earlier list linodes
		// Need sub command options if no options default to create linode if list or delete subcommand or choice list

		/*
			 {
			  "alerts": {
			    "cpu": 90,
			    "io": 10000,
			    "network_in": 10,
			    "network_out": 10,
			    "transfer_quota": 80
			  },
			  "backups": {
			    "available": false,
			    "enabled": false,
			    "last_successful": null,
			    "schedule": {
			      "day": null,
			      "window": null
			    }
			  },
			  "capabilities": [],
			  "created": "2025-07-18T02:56:19",
			  "disk_encryption": "enabled",
			  "group": "",
			  "has_user_data": false,
			  "host_uuid": "b2fa42934d38726a3bfae96212b57681f45b7eec",
			  "hypervisor": "kvm",
			  "id": 80368586,
			  "image": "linode/almalinux8",
			  "ipv4": [
			    "194.195.116.217"
			  ],
			  "ipv6": "2400:8904::2000:a7ff:fee4:c178/128",
			  "label": "linode80368586",
			  "lke_cluster_id": null,
			  "placement_group": null,
			  "region": "ap-west",
			  "site_type": "core",
			  "specs": {
			    "accelerated_devices": 0,
			    "disk": 25600,
			    "gpus": 0,
			    "memory": 1024,
			    "transfer": 1000,
			    "vcpus": 1
			  },
			  "status": "provisioning",
			  "tags": [],
			  "type": "g6-nanode-1",
			  "updated": "2025-07-18T02:56:19",
			  "watchdog_enabled": true
			}
		*/

	case "DigitalOcean":
		logger.Info("Work in progress...")
	case "Vultr":
		logger.Info("Work in progress...")
	default:
		logger.Warn("No provider was selected. Exiting...")
	}
}
