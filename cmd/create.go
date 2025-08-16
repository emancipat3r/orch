package cmd

import (
	"strings"

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
			providerKey := providers.GetLinodeAPIKey(configFile, provider)
			accountBalance, err := providers.GetDOBalance(providerKey)

			if err != nil {
				logger.Error("Failed to get DigitalOcean account balance: " + err.Error())
				return
			}

			logger.Info("DigitalOcean account balance: " + logger.Highlight("$"+accountBalance))

			regions, err := providers.GetDORegions(providerKey)

			var regionOptions []string

			for _, region := range regions {
				regionOptions = append(regionOptions, region.Slug+" - "+region.Name)
			}

			selectedRegion := ui.Select("Select your region:", regionOptions)
			logger.Info("You selected region: " + logger.Highlight(selectedRegion))
			//selectedRegionSplit := strings.Split(selectedRegion, " ")

		case "Vultr":
			logger.Info("Work in progress...")
		default:
			logger.Warn("No provider was selected. Exiting...")
		}
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
}
