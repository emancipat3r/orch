package cmd

import (
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/charmbracelet/keygen"
	"github.com/emancipat3r/vps3/logger"
	"github.com/emancipat3r/vps3/providers"
	"github.com/emancipat3r/vps3/ui"
	"github.com/emancipat3r/vps3/utils"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Provision a new VPS instance",
	Run: func(cmd *cobra.Command, args []string) {
		// Set up signal handling for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigChan
			logger.Info("\nOperation cancelled by user (Ctrl+C)")
			os.Exit(0)
		}()

		// Use wizard interface for provider selection and navigation
		provider := ui.ChoiceProvider()
		if provider == "" {
			logger.Info("Operation cancelled by user.")
			return
		}
		logger.Info("You selected: " + logger.Highlight(provider))

		// Load data for the selected provider and create navigable wizard
		switch provider {
		case "Linode":
			providerKey := providers.GetLinodeAPIKey(configFile, provider)
			accountBalance, err := providers.GetLinodesBalance(providerKey)

			if err != nil {
				logger.Error("Failed to get Linode account balance: " + err.Error())
				return
			}

			logger.Info("Linode account balance: " + logger.Highlight("$"+accountBalance))

			regions, err := providers.GetLinodeRegions()

			if err != nil {
				logger.Error("Failed to hit endpoint: " + err.Error())
				return
			}

			var regionOptions []string

			// Load all options for Linode
			for _, region := range regions {
				if region.Status == "ok" {
					regionOptions = append(regionOptions, region.ID+" - "+region.Label)
				}
			}

			images, err := providers.GetLinodeImages()
			if err != nil {
				logger.Error("Failed to get Linode images: " + err.Error())
				return
			}
			var imageOptions []string
			for _, image := range images {
				imageOptions = append(imageOptions, image.ID+" - "+image.Label)
			}

			resources, err := providers.GetLinodeResources()
			if err != nil {
				logger.Error("Failed to get Linode resources: " + err.Error())
				return
			}
			var resourceOptions []string
			for _, resource := range resources {
				resourceOptions = append(resourceOptions, resource.ID+" - "+resource.Label)
			}

			// Create navigable wizard with all options loaded
			var region, image, size string
			steps := []ui.WizardStep{
				{
					Key:         "region",
					Title:       "Select your region",
					Description: "",
					Options:     regionOptions,
					Value:       &region,
				},
				{
					Key:         "image",
					Title:       "Select your image",
					Description: "",
					Options:     imageOptions,
					Value:       &image,
				},
				{
					Key:         "size",
					Title:       "Select your resourcing",
					Description: "",
					Options:     resourceOptions,
					Value:       &size,
				},
			}

			selections, err := ui.CreateNavigableWizard(steps)
			if err != nil {
				logger.Info("Operation cancelled by user.")
				return
			}

			selectedRegion := selections["region"]
			selectedImage := selections["image"]
			selectedResource := selections["size"]

			logger.Info("You selected region: " + logger.Highlight(selectedRegion))
			logger.Info("You selected image: " + logger.Highlight(selectedImage))
			logger.Info("You selected type: " + logger.Highlight(selectedResource))

			selectedRegionSplit := strings.Split(selectedRegion, " ")
			selectedImageSplit := strings.Split(selectedImage, " ")
			selectedResourceSplit := strings.Split(selectedResource, " ")

			rootPassword, err := utils.GenerateRandomPassword(30)

			if err != nil {
				logger.Error("Failed to generate root password for Linode: " + err.Error())
				return
			}

			// === Per-instance keypair (using charmbracelet/keygen) ===
			keyName := "linode-" + providers.CreateUID()
			perPriv := pathSSH + keyName // ~/.config/vps/.ssh/linode-XXXXXXXX
			perPub := perPriv + ".pub"

			// optional: passphrase (store per key)
			pass, err := utils.GenerateRandomPassword(24)
			if err != nil {
				logger.Error("Failed to gen passphrase: " + err.Error())
				return
			}
			if err := os.WriteFile(pathSecrets+keyName+".pass", []byte(pass+"\n"), 0600); err != nil {
				logger.Error("Failed to store passphrase: " + err.Error())
				return
			}

			// create the keypair
			if _, err := keygen.New(perPriv, keygen.WithKeyType(keygen.Ed25519), keygen.WithPassphrase(pass), keygen.WithWrite()); err != nil {
				logger.Error("Failed to create per-instance keypair: " + err.Error())
				return
			}

			// Add key to ssh-agent if available
			if err := utils.AddKeyToSSHAgent(perPriv); err != nil {
				logger.Debug("Could not add key to ssh-agent: " + err.Error())
			}

			// Upload pubkey to Linode → get unique key ID
			keyID, err := providers.UploadLinodeSSHKey(providerKey, perPub)
			if err != nil {
				logger.Error("Failed to upload SSH key to Linode: " + err.Error())
				return
			}
			logger.Info("Per-instance SSH key ID: " + logger.Highlight(strconv.Itoa(keyID)))

			// Create the instance with this key ID and persist priv key path
			logger.Info("Creating Linode...")
			_, err = providers.CreateLinode(
				providerKey,
				keyID,
				perPriv,
				selectedImageSplit[0],
				selectedRegionSplit[0],
				selectedResourceSplit[0],
				rootPassword,
				instanceFile,
			)

			if err != nil {
				logger.Error("Failed to create Linode: " + err.Error())
			}

		case "DigitalOcean":
			providerKey := providers.GetDOAPIKey(configFile, provider)
			accountBalance, err := providers.GetDOBalance(providerKey)
			if err != nil {
				logger.Error("Failed to get DigitalOcean account balance: " + err.Error())
				return
			}
			logger.Info("DigitalOcean account balance: " + logger.Highlight("$"+accountBalance))

			// region
			regions, err := providers.GetDORegions(providerKey)
			if err != nil {
				logger.Error("Failed to hit endpoint: " + err.Error())
				return
			}
			var regionOptions []string
			for _, r := range regions {
				regionOptions = append(regionOptions, r.Slug+" - "+r.Name)
			}
			// Load all options for DigitalOcean
			images, err := providers.GetDOImages(providerKey)
			if err != nil {
				logger.Error("Failed to get DigitalOcean images: " + err.Error())
				return
			}
			var imageOptions []string
			for _, img := range images {
				imageOptions = append(imageOptions, img.Slug+" - "+img.Name)
			}

			sizes, err := providers.GetDOResources(providerKey)
			if err != nil {
				logger.Error("Failed to get DigitalOcean sizes: " + err.Error())
				return
			}
			var resourceOptions []string
			for _, s := range sizes {
				resourceOptions = append(resourceOptions, s.Slug+" - "+s.Name)
			}

			// Create navigable wizard with all options loaded
			var region, image, size string
			steps := []ui.WizardStep{
				{
					Key:         "region",
					Title:       "Select your region",
					Description: "",
					Options:     regionOptions,
					Value:       &region,
				},
				{
					Key:         "image",
					Title:       "Select your image",
					Description: "",
					Options:     imageOptions,
					Value:       &image,
				},
				{
					Key:         "size",
					Title:       "Select your resourcing",
					Description: "",
					Options:     resourceOptions,
					Value:       &size,
				},
			}

			selections, err := ui.CreateNavigableWizard(steps)
			if err != nil {
				logger.Info("Operation cancelled by user.")
				return
			}

			selectedRegion := selections["region"]
			selectedImage := selections["image"]
			selectedResource := selections["size"]

			logger.Info("You selected region: " + logger.Highlight(selectedRegion))
			logger.Info("You selected image: " + logger.Highlight(selectedImage))
			logger.Info("You selected type: " + logger.Highlight(selectedResource))

			selectedRegionSlug := strings.Split(selectedRegion, " ")[0]
			selectedImageSlug := strings.Split(selectedImage, " ")[0]
			selectedSizeSlug := strings.Split(selectedResource, " ")[0]

			// === Per-droplet keypair (using charmbracelet/keygen) ===
			keyName := "do-" + providers.CreateUID()
			perPriv := pathSSH + keyName // ~/.config/vps/.ssh/do-XXXXXXXX
			perPub := perPriv + ".pub"

			// optional: passphrase (store per key)
			pass, err := utils.GenerateRandomPassword(24)
			if err != nil {
				logger.Error("Failed to gen passphrase: " + err.Error())
				return
			}
			if err := os.WriteFile(pathSecrets+keyName+".pass", []byte(pass+"\n"), 0600); err != nil {
				logger.Error("Failed to store passphrase: " + err.Error())
				return
			}

			// create the keypair
			if _, err := keygen.New(perPriv, keygen.WithKeyType(keygen.Ed25519), keygen.WithPassphrase(pass), keygen.WithWrite()); err != nil {
				logger.Error("Failed to create per-droplet keypair: " + err.Error())
				return
			}

			// Add key to ssh-agent if available
			if err := utils.AddKeyToSSHAgent(perPriv); err != nil {
				logger.Debug("Could not add key to ssh-agent: " + err.Error())
			}

			// Upload pubkey to DigitalOcean → get unique key ID
			keyID, err := providers.UploadDOSSHKey(providerKey, perPub)
			if err != nil {
				logger.Error("Failed to upload SSH key to DigitalOcean: " + err.Error())
				return
			}
			logger.Info("Droplet SSH key ID: " + logger.Highlight(perPriv))

			// Create the droplet with this key ID and persist priv key path
			_, err = providers.CreateDroplet(
				providerKey,
				keyID,
				perPriv,
				selectedImageSlug,
				selectedRegionSlug,
				selectedSizeSlug,
				instanceFile,
			)

			if err != nil {
				logger.Error("Failed to create Droplet: " + err.Error())
			}

		case "Vultr":
			providerKey := providers.GetVultrAPIKey(configFile, provider)
			accountBalance, err := providers.GetVultrBalance(providerKey)
			if err != nil {
				logger.Error("Failed to get Vultr account balance: " + err.Error())
				return
			}
			logger.Info("Vultr account balance: " + logger.Highlight("$"+accountBalance))

			// region
			regions, err := providers.GetVultrRegions(providerKey)
			if err != nil {
				logger.Error("Failed to hit endpoint: " + err.Error())
				return
			}
			var regionOptions []string
			for _, r := range regions {
				regionOptions = append(regionOptions, r.ID+" - "+r.City+", "+r.Country)
			}
			// Load all options for Vultr
			osImages, err := providers.GetVultrOS(providerKey)
			if err != nil {
				logger.Error("Failed to get Vultr OS images: " + err.Error())
				return
			}
			var osOptions []string
			for _, os := range osImages {
				osOptions = append(osOptions, strconv.Itoa(os.ID)+" - "+os.Name)
			}

			plans, err := providers.GetVultrPlans(providerKey)
			if err != nil {
				logger.Error("Failed to get Vultr plans: " + err.Error())
				return
			}
			var planOptions []string
			for _, p := range plans {
				planOptions = append(planOptions, p.ID+" - "+strconv.Itoa(p.VCPUCount)+" vCPU, "+strconv.Itoa(p.RAM)+" MB RAM, "+strconv.Itoa(p.Disk)+" GB SSD - $"+strconv.FormatFloat(p.MonthlyCost, 'f', 2, 64)+"/month")
			}

			// Create navigable wizard with all options loaded
			var region, os, plan string
			steps := []ui.WizardStep{
				{
					Key:         "region",
					Title:       "Select your region",
					Description: "",
					Options:     regionOptions,
					Value:       &region,
				},
				{
					Key:         "os",
					Title:       "Select your OS",
					Description: "",
					Options:     osOptions,
					Value:       &os,
				},
				{
					Key:         "plan",
					Title:       "Select your plan",
					Description: "",
					Options:     planOptions,
					Value:       &plan,
				},
			}

			selections, err := ui.CreateNavigableWizard(steps)
			if err != nil {
				logger.Info("Operation cancelled by user.")
				return
			}

			selectedRegion := selections["region"]
			selectedOS := selections["os"]
			selectedPlan := selections["plan"]

			logger.Info("You selected region: " + logger.Highlight(selectedRegion))
			logger.Info("You selected OS: " + logger.Highlight(selectedOS))
			logger.Info("You selected plan: " + logger.Highlight(selectedPlan))

			selectedRegionID := strings.Split(selectedRegion, " ")[0]
			selectedOSIDStr := strings.Split(selectedOS, " ")[0]
			selectedOSID, _ := strconv.Atoi(selectedOSIDStr)
			selectedPlanID := strings.Split(selectedPlan, " ")[0]

			// === Per-instance keypair (using charmbracelet/keygen) ===
			keyName := "vultr-" + providers.CreateUID()
			perPriv := pathSSH + keyName // ~/.config/vps/.ssh/vultr-XXXXXXXX
			perPub := perPriv + ".pub"

			// optional: passphrase (store per key)
			pass, err := utils.GenerateRandomPassword(24)
			if err != nil {
				logger.Error("Failed to gen passphrase: " + err.Error())
				return
			}
			if err := ioutil.WriteFile(pathSecrets+keyName+".pass", []byte(pass+"\n"), 0600); err != nil {
				logger.Error("Failed to store passphrase: " + err.Error())
				return
			}

			// create the keypair
			if _, err := keygen.New(perPriv, keygen.WithKeyType(keygen.Ed25519), keygen.WithPassphrase(pass), keygen.WithWrite()); err != nil {
				logger.Error("Failed to create per-instance keypair: " + err.Error())
				return
			}

			// Add key to ssh-agent if available
			if err := utils.AddKeyToSSHAgent(perPriv); err != nil {
				logger.Debug("Could not add key to ssh-agent: " + err.Error())
			}

			// Upload pubkey to Vultr → get unique key ID
			keyID, err := providers.UploadVultrSSHKey(providerKey, perPub)
			if err != nil {
				logger.Error("Failed to upload SSH key to Vultr: " + err.Error())
				return
			}
			logger.Info("Per-instance SSH key ID: " + logger.Highlight(keyID))

			// Create the instance with this key ID and persist priv key path
			logger.Info("Creating Vultr instance...")
			instanceID, err := providers.CreateVultrInstance(
				providerKey,
				keyID,
				perPriv,
				selectedOSID,
				selectedRegionID,
				selectedPlanID,
				instanceFile,
			)

			if err != nil {
				logger.Error("Failed to create Vultr instance: " + err.Error())
			} else {
				logger.Info("Successfully created Vultr instance: " + logger.Highlight(instanceID))
			}

		default:
			logger.Warn("No provider was selected. Exiting...")
		}
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
}
