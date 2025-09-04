package cmd

import (
	"os"
	"os/user"
	"strconv"
	"strings"

	"github.com/emancipat3r/vps3/logger"
	"github.com/emancipat3r/vps3/providers"
	"github.com/emancipat3r/vps3/ui"
	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy a VPS instance",
	Run: func(cmd *cobra.Command, args []string) {
		provider := ui.ChoiceProvider()
		user, err := user.Current()
		if err != nil {
			logger.Error("Failed to get current user: " + err.Error())
			return
		}

		pathConfig = user.HomeDir + "/.config/vps/config/"
		configFile = pathConfig + "configuration.toml"
		pathInstances = user.HomeDir + "/.config/vps/instances/"
		instanceFile = pathInstances + "instances.toml"

		switch provider {
		case "Linode":
			providerKey := providers.GetLinodeAPIKey(configFile, provider)

			accountBalance, err := providers.GetLinodesBalance(providerKey)

			if err != nil {
				logger.Error("Failed to get Linode account balance: " + err.Error())
				return
			}

			logger.Info("Linode account balance: " + logger.Highlight("$"+accountBalance))

			instances, err := providers.SelectLinodeInstance(providerKey)

			if err != nil {
				logger.Error("Failed to hit endpoint: " + err.Error())
				return
			}

			var instanceOptions []string

			for _, instance := range instances {
				instanceOptions = append(instanceOptions, instance.Creation_Time+" - "+strconv.Itoa(instance.Id)+" - "+instance.Host_Image+" - "+instance.Ipv4[0]+" - "+instance.Region)
			}

			selectedInstance := ui.Select("Select the instance:", instanceOptions)
			logger.Info("You selected instance: " + logger.Highlight(selectedInstance))
			choice := ui.Confirm("Are you sure you want to proceed?")

			if choice == false {
				os.Exit(1)
			}

			selectedInstanceSplit := strings.Split(selectedInstance, " ")

			logger.Info("Destroying Linode: " + logger.Highlight(selectedInstanceSplit[2]))

			providers.DestroyLinode(providerKey, selectedInstanceSplit[2], instanceFile)

			if err != nil {
				logger.Error("Failed to destroy Linode: " + err.Error())
			}

		case "DigitalOcean":
			providerKey := providers.GetDOAPIKey(configFile, provider)
			accountBalance, err := providers.GetDOBalance(providerKey)
			if err != nil {
				logger.Error("Failed to get DigitalOcean account balance: " + err.Error())
				return
			}
			logger.Info("DigitalOcean account balance: " + logger.Highlight("$"+accountBalance))

			instances, err := providers.SelectDOInstance(providerKey)

			if err != nil {
				logger.Error("Failed to hit endpoint: " + err.Error())
				return
			}

			var instanceOptions []string

			for _, instance := range instances {
				var ipv4 string
				// Find the public IPv4 address
				for _, v4 := range instance.Networks.IPv4 {
					if v4.Type == "public" {
						ipv4 = v4.IPAddress
						break
					}
				}

				// Use image description if available, fallback to name, then slug
				imageName := instance.Image.Description
				if imageName == "" {
					imageName = instance.Image.Name
				}
				if imageName == "" {
					imageName = instance.Image.Slug
				}

				instanceOptions = append(instanceOptions, instance.Creation_Time+" - "+strconv.Itoa(instance.Id)+" - "+imageName+" - "+ipv4+" - "+instance.Region.Slug)
			}

			selectedInstance := ui.Select("Select the instance:", instanceOptions)
			logger.Info("You selected instance: " + logger.Highlight(selectedInstance))
			choice := ui.Confirm("Are you sure you want to proceed?")

			if choice == false {
				os.Exit(1)
			}

			selectedInstanceSplit := strings.Split(selectedInstance, " ")

			logger.Info("Destroying droplet: " + logger.Highlight(selectedInstanceSplit[2]))

			providers.DestroyDroplet(providerKey, selectedInstanceSplit[2], instanceFile)

			if err != nil {
				logger.Error("Failed to destroy droplet: " + err.Error())
			}

		case "Vultr":
			providerKey := providers.GetVultrAPIKey(configFile, provider)
			accountBalance, err := providers.GetVultrBalance(providerKey)
			if err != nil {
				logger.Error("Failed to get Vultr account balance: " + err.Error())
				return
			}
			logger.Info("Vultr account balance: " + logger.Highlight("$"+accountBalance))

			instances, err := providers.SelectVultrInstance(providerKey)

			if err != nil {
				logger.Error("Failed to hit endpoint: " + err.Error())
				return
			}

			var instanceOptions []string

			for _, instance := range instances {
				instanceOptions = append(instanceOptions, instance.DateCreated+" - "+instance.ID+" - "+instance.OS+" - "+instance.MainIP+" - "+instance.Region)
			}

			selectedInstance := ui.Select("Select the instance:", instanceOptions)
			logger.Info("You selected instance: " + logger.Highlight(selectedInstance))
			choice := ui.Confirm("Are you sure you want to proceed?")

			if choice == false {
				os.Exit(1)
			}

			selectedInstanceSplit := strings.Split(selectedInstance, " ")

			logger.Info("Destroying Vultr instance: " + logger.Highlight(selectedInstanceSplit[2]))

			err = providers.DestroyVultr(providerKey, selectedInstanceSplit[2], instanceFile)

			if err != nil {
				logger.Error("Failed to destroy Vultr instance: " + err.Error())
			}
		default:
			logger.Warn("No provider was selected. Exiting...")
		}
	},
}

func init() {
	rootCmd.AddCommand(destroyCmd)
}
