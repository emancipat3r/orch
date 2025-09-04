package cmd

import (
	"os"
	"strconv"
	"strings"

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
		provider := ui.ChoiceProvider()
		logger.Info("You selected: " + logger.Highlight(provider))

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

			for _, region := range regions {
				if region.Status == "ok" {
					regionOptions = append(regionOptions, region.ID+" - "+region.Label)
				}
			}

			selectedRegion := ui.Select("Select your region:", regionOptions)
			logger.Info("You selected region: " + logger.Highlight(selectedRegion))
			selectedRegionSplit := strings.Split(selectedRegion, " ")

			images, err := providers.GetLinodeImages()
			if err != nil {
				logger.Error("Failed to hit endpoint: " + err.Error())
				return
			}

			var imageOptions []string

			for _, image := range images {
				imageOptions = append(imageOptions, image.ID+" - "+image.Label)
			}

			selectedImage := ui.Select("Select your image:", imageOptions)
			logger.Info("You selected image: " + logger.Highlight(selectedImage))
			selectedImageSplit := strings.Split(selectedImage, " ")

			resources, err := providers.GetLinodeResources()
			if err != nil {
				logger.Error("Failed to hit endpoint: " + err.Error())
				return
			}

			var resourceOptions []string

			for _, resource := range resources {
				resourceOptions = append(resourceOptions, resource.ID+" - "+resource.Label)
			}

			selectedResource := ui.Select("Select your resourcing:", resourceOptions)
			logger.Info("You selected type: " + logger.Highlight(selectedResource))
			selectedResourceSplit := strings.Split(selectedResource, " ")

			rootPassword, err := utils.GenerateRandomPassword(30)

			if err != nil {
				logger.Error("Failed to generate root password for Linode: " + err.Error())
				return
			}

			logger.Info("Creating Linode...")
			_, err = providers.CreateLinode(
				providerKey,
				publicKeyPath,
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
			selectedRegion := ui.Select("Select your region:", regionOptions)
			logger.Info("You selected region: " + logger.Highlight(selectedRegion))
			selectedRegionSlug := strings.Split(selectedRegion, " ")[0]

			// image
			images, err := providers.GetDOImages(providerKey)
			if err != nil {
				logger.Error("Failed to hit endpoint: " + err.Error())
				return
			}
			var imageOptions []string
			for _, img := range images {
				imageOptions = append(imageOptions, img.Slug+" - "+img.Name)
			}
			selectedImage := ui.Select("Select your image:", imageOptions)
			logger.Info("You selected image: " + logger.Highlight(selectedImage))
			selectedImageSlug := strings.Split(selectedImage, " ")[0]

			// size
			resources, err := providers.GetDOResources(providerKey)
			if err != nil {
				logger.Error("Failed to hit endpoint: " + err.Error())
				return
			}
			var resourceOptions []string
			for _, s := range resources {
				resourceOptions = append(resourceOptions, s.Slug+" - "+s.Name)
			}
			selectedResource := ui.Select("Select your resourcing:", resourceOptions)
			logger.Info("You selected type: " + logger.Highlight(selectedResource))
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

			// Upload pubkey to DO → get unique key ID
			keyID, err := providers.UploadDOSSHKey(providerKey, perPub)
			if err != nil {
				logger.Error("Failed to upload SSH key to DigitalOcean: " + err.Error())
				return
			}
			logger.Info("Per-droplet SSH key ID: " + logger.Highlight(strconv.Itoa(keyID)))

			// Create the droplet with this key ID and persist priv key path
			logger.Info("Creating Droplet...")
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
			selectedRegion := ui.Select("Select your region:", regionOptions)
			logger.Info("You selected region: " + logger.Highlight(selectedRegion))
			selectedRegionID := strings.Split(selectedRegion, " ")[0]

			// OS images
			osImages, err := providers.GetVultrOS(providerKey)
			if err != nil {
				logger.Error("Failed to hit endpoint: " + err.Error())
				return
			}
			var osOptions []string
			for _, os := range osImages {
				osOptions = append(osOptions, strconv.Itoa(os.ID)+" - "+os.Name)
			}
			selectedOS := ui.Select("Select your OS:", osOptions)
			logger.Info("You selected OS: " + logger.Highlight(selectedOS))
			selectedOSIDStr := strings.Split(selectedOS, " ")[0]
			selectedOSID, _ := strconv.Atoi(selectedOSIDStr)

			// plans
			plans, err := providers.GetVultrPlans(providerKey)
			if err != nil {
				logger.Error("Failed to hit endpoint: " + err.Error())
				return
			}
			var planOptions []string
			for _, p := range plans {
				// Filter plans available in the selected region
				regionAvailable := false
				for _, loc := range p.Locations {
					if loc == selectedRegionID {
						regionAvailable = true
						break
					}
				}
				if regionAvailable {
					planOptions = append(planOptions, p.ID+" - $"+strconv.FormatFloat(p.MonthlyCost, 'f', 2, 64)+"/mo ("+strconv.Itoa(p.VCPUCount)+" vCPU, "+strconv.Itoa(p.RAM)+"MB RAM)")
				}
			}
			selectedPlan := ui.Select("Select your plan:", planOptions)
			logger.Info("You selected plan: " + logger.Highlight(selectedPlan))
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
			if err := os.WriteFile(pathSecrets+keyName+".pass", []byte(pass+"\n"), 0600); err != nil {
				logger.Error("Failed to store passphrase: " + err.Error())
				return
			}

			// create the keypair
			if _, err := keygen.New(perPriv, keygen.WithKeyType(keygen.Ed25519), keygen.WithPassphrase(pass), keygen.WithWrite()); err != nil {
				logger.Error("Failed to create per-instance keypair: " + err.Error())
				return
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
