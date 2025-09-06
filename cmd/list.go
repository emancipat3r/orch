package cmd

import (
	"os/user"

	"github.com/emancipat3r/vps3/logger"
	"github.com/emancipat3r/vps3/providers"
	"github.com/emancipat3r/vps3/ui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List running VPS instances",
	Run: func(cmd *cobra.Command, args []string) {
		provider := ui.ChoiceProvider()

		switch provider {
		case "Linode":
			user, err := user.Current()
			if err != nil {
				logger.Error("Failed to get current user: " + err.Error())
				return
			}

			pathConfig = user.HomeDir + "/.config/vps/config/"
			configFile = pathConfig + "configuration.toml"
			pathInstances = user.HomeDir + "/.config/vps/instances/"
			instanceFile = pathInstances + "instances.toml"

			providerKey := providers.GetLinodeAPIKey(configFile, provider)

			accountBalance, err := providers.GetLinodesBalance(providerKey)

			if err != nil {
				logger.Error("Failed to get Linode account balance: " + err.Error())
				return
			}

			logger.Info("Linode account balance: " + logger.Highlight("$"+accountBalance))

			providers.ListLinodeInstancesTable(providerKey, instanceFile)

		case "DigitalOcean":
			user, err := user.Current()
			if err != nil {
				logger.Error("Failed to get current user: " + err.Error())
				return
			}

			pathConfig = user.HomeDir + "/.config/vps/config/"
			configFile = pathConfig + "configuration.toml"

			providerKey := providers.GetDOAPIKey(configFile, provider)

			accountBalance, err := providers.GetDOBalance(providerKey)

			if err != nil {
				logger.Error("Failed to get DigitalOcean account balance: " + err.Error())
				return
			}

			logger.Info("DigitalOcean account balance: " + logger.Highlight("$"+accountBalance))

			_, err = providers.ListDOInstancesTable(providerKey)
			if err != nil {
				logger.Error("Failed to list DigitalOcean instances: " + err.Error())
			}
		case "Vultr":
			user, err := user.Current()
			if err != nil {
				logger.Error("Failed to get current user: " + err.Error())
				return
			}

			pathConfig = user.HomeDir + "/.config/vps/config/"
			configFile = pathConfig + "configuration.toml"
			pathInstances = user.HomeDir + "/.config/vps/instances/"
			instanceFile = pathInstances + "instances.toml"

			providerKey := providers.GetVultrAPIKey(configFile, provider)

			accountBalance, err := providers.GetVultrBalance(providerKey)

			if err != nil {
				logger.Error("Failed to get Vultr account balance: " + err.Error())
				return
			}

			logger.Info("Vultr account balance: " + logger.Highlight("$"+accountBalance))

			_, err = providers.ListVultrInstancesTable(providerKey, instanceFile)
			if err != nil {
				logger.Error("Failed to list Vultr instances: " + err.Error())
			}
		default:
			logger.Warn("No provider was selected. Exiting...")
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
