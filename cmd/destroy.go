package cmd

import (
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"strings"
	"syscall"

	"github.com/emancipat3r/vps3/logger"
	"github.com/emancipat3r/vps3/providers"
	"github.com/emancipat3r/vps3/ui"
	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy VPS instance(s)",
	Run: func(cmd *cobra.Command, args []string) {
		// Set up signal handling for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigChan
			logger.Info("\nOperation cancelled by user (Ctrl+C)")
			os.Exit(0)
		}()

		provider := ui.ChoiceProvider()
		if provider == "" {
			logger.Info("Operation cancelled by user.")
			return
		}
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

			selectedInstances := ui.MultiSelect("Select the instance(s) to destroy:", instanceOptions)
			if len(selectedInstances) == 0 {
				logger.Info("No instances selected. Exiting...")
				return
			}

			logger.Info("You selected " + strconv.Itoa(len(selectedInstances)) + " instance(s) for destruction")
			for _, instance := range selectedInstances {
				logger.Info("  - " + logger.Highlight(instance))
			}

			choice := ui.Confirm("Are you sure you want to proceed with destroying " + strconv.Itoa(len(selectedInstances)) + " instance(s)?")

			if !choice {
				logger.Info("Destruction cancelled by user.")
				return
			}

			for i, selectedInstance := range selectedInstances {
				selectedInstanceSplit := strings.Split(selectedInstance, " ")
				instanceID := selectedInstanceSplit[2]

				logger.Info("(" + strconv.Itoa(i+1) + "/" + strconv.Itoa(len(selectedInstances)) + ") Destroying Linode: " + logger.Highlight(instanceID))

				_, err = providers.DestroyLinode(providerKey, instanceID, instanceFile)

				if err != nil {
					logger.Error("Failed to destroy Linode " + instanceID + ": " + err.Error())
				} else {
					logger.Info("Successfully destroyed Linode: " + logger.Highlight(instanceID))
				}
			}

			logger.Info("Completed destruction of " + strconv.Itoa(len(selectedInstances)) + " Linode instance(s)")

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

			selectedInstances := ui.MultiSelect("Select the instance(s) to destroy:", instanceOptions)
			if len(selectedInstances) == 0 {
				logger.Info("No instances selected. Exiting...")
				return
			}

			logger.Info("You selected " + strconv.Itoa(len(selectedInstances)) + " instance(s) for destruction")
			for _, instance := range selectedInstances {
				logger.Info("  - " + logger.Highlight(instance))
			}

			choice := ui.Confirm("Are you sure you want to proceed with destroying " + strconv.Itoa(len(selectedInstances)) + " instance(s)?")

			if !choice {
				logger.Info("Destruction cancelled by user.")
				return
			}

			for i, selectedInstance := range selectedInstances {
				selectedInstanceSplit := strings.Split(selectedInstance, " ")
				instanceID := selectedInstanceSplit[2]

				logger.Info("(" + strconv.Itoa(i+1) + "/" + strconv.Itoa(len(selectedInstances)) + ") Destroying droplet: " + logger.Highlight(instanceID))

				_, err = providers.DestroyDroplet(providerKey, instanceID, instanceFile)

				if err != nil {
					logger.Error("Failed to destroy droplet " + instanceID + ": " + err.Error())
				} else {
					logger.Info("Successfully destroyed droplet: " + logger.Highlight(instanceID))
				}
			}

			logger.Info("Completed destruction of " + strconv.Itoa(len(selectedInstances)) + " DigitalOcean instance(s)")

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

			regions, err := providers.GetVultrRegions(providerKey)
			regionCache := make(map[string]string)
			if err == nil {
				for _, region := range regions {
					regionCache[region.ID] = region.ID + " - " + region.City + ", " + region.Country
				}
			}

			var instanceOptions []string

			for _, instance := range instances {
				regionDisplay := instance.Region
				if verboseRegion, exists := regionCache[instance.Region]; exists {
					regionDisplay = verboseRegion
				}
				instanceOptions = append(instanceOptions, instance.DateCreated+" - "+instance.ID+" - "+instance.OS+" - "+instance.MainIP+" - "+regionDisplay)
			}

			selectedInstances := ui.MultiSelect("Select the instance(s) to destroy:", instanceOptions)
			if len(selectedInstances) == 0 {
				logger.Info("No instances selected. Exiting...")
				return
			}

			logger.Info("You selected " + strconv.Itoa(len(selectedInstances)) + " instance(s) for destruction")
			for _, instance := range selectedInstances {
				logger.Info("  - " + logger.Highlight(instance))
			}

			choice := ui.Confirm("Are you sure you want to proceed with destroying " + strconv.Itoa(len(selectedInstances)) + " instance(s)?")

			if !choice {
				logger.Info("Destruction cancelled by user.")
				return
			}

			for i, selectedInstance := range selectedInstances {
				selectedInstanceSplit := strings.Split(selectedInstance, " ")
				instanceID := selectedInstanceSplit[2]

				logger.Info("(" + strconv.Itoa(i+1) + "/" + strconv.Itoa(len(selectedInstances)) + ") Destroying Vultr instance: " + logger.Highlight(instanceID))

				err = providers.DestroyVultr(providerKey, instanceID, instanceFile)

				if err != nil {
					logger.Error("Failed to destroy Vultr instance " + instanceID + ": " + err.Error())
				} else {
					logger.Info("Successfully destroyed Vultr instance: " + logger.Highlight(instanceID))
				}
			}

			logger.Info("Completed destruction of " + strconv.Itoa(len(selectedInstances)) + " Vultr instance(s)")
		default:
			logger.Warn("No provider was selected. Exiting...")
		}
	},
}

func init() {
	rootCmd.AddCommand(destroyCmd)
}
