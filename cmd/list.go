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

			providerKey := providers.GetLinodeAPIKey(configFile, provider)

			accountBalance, err := providers.GetLinodesBalance(providerKey)

			if err != nil {
				logger.Error("Failed to get Linode account balance: " + err.Error())
				return
			}

			logger.Info("Linode account balance: " + logger.Highlight("$"+accountBalance))

			providers.ListLinodeInstancesTable(providerKey)

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

			providers.ListDOInstancesTable(providerKey)
		case "Vultr":
			logger.Info("Work in progress...")
		default:
			logger.Warn("No provider was selected. Exiting...")
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
