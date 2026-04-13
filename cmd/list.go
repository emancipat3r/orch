package cmd

import (
	"os"
	"os/signal"
	"os/user"
	"syscall"

	"github.com/emancipat3r/orch/logger"
	"github.com/emancipat3r/orch/providers"
	"github.com/emancipat3r/orch/ui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List running VPS instances",
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

		switch provider {
		case "Linode":
			user, err := user.Current()
			if err != nil {
				logger.Error("Failed to get current user: " + err.Error())
				return
			}

			pathConfig = user.HomeDir + "/.config/orch/config/"
			configFile = pathConfig + "configuration.toml"
			pathInstances = user.HomeDir + "/.config/orch/instances/"
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

			pathConfig = user.HomeDir + "/.config/orch/config/"
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

			pathConfig = user.HomeDir + "/.config/orch/config/"
			configFile = pathConfig + "configuration.toml"
			pathInstances = user.HomeDir + "/.config/orch/instances/"
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
