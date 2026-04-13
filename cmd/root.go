package cmd

import (
	"fmt"
	"os"
	"os/user"

	"github.com/emancipat3r/orch/logger"
	"github.com/emancipat3r/orch/utils"
	"github.com/spf13/cobra"
)

var (
	pathConfig    string
	pathSSH       string
	pathSecrets   string
	pathInstances string
	pathWg        string
	instanceFile  string
	configFile    string
	pathAnsible   string
	vpsName       string
)

// rootCmd is the base command for the CLI
var rootCmd = &cobra.Command{
	Use:   "orch",
	Short: "VPS management CLI",
	Long:  "A CLI tool for provisioning and managing VPS instances.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Always set path variables
		user, err := user.Current()
		if err != nil {
			logger.Error("Failed to get current user: " + err.Error())
			return err
		}

		// Skip all *other* checks for these commands
		skipChecks := map[string]bool{"help": true, "completion": true, "list": true, "destroy": true}
		if skipChecks[cmd.Name()] {
			return nil
		}

		pathConfig = user.HomeDir + "/.config/orch/config/"
		pathSSH = user.HomeDir + "/.config/orch/.ssh/"
		pathSecrets = user.HomeDir + "/.config/orch/secrets/"
		pathInstances = user.HomeDir + "/.config/orch/instances/"
		pathAnsible = user.HomeDir + "/.config/orch/ansible/"
		pathWg = user.HomeDir + "/.config/orch/wg/"
		instanceFile = pathInstances + "instances.toml"
		configFile = pathConfig + "configuration.toml"

		// Ensure all required directories exist
		for _, dir := range []string{pathConfig, pathSSH, pathSecrets, pathInstances, pathAnsible, pathWg} {
			if !utils.DirExists(dir) {
				logger.Warn(fmt.Sprintf("Directory doesn't exist. Creating: %s", logger.Highlight(dir)))
				if err := utils.MakeDirectory(dir); err != nil {
					logger.Error("Failed to create directory: " + err.Error())
					return err
				}
				logger.Info("Directory created: " + logger.Highlight(dir))
			} else {
				logger.Info("Directory exists: " + logger.Highlight(dir))
			}
		}

		// Check for credentials (config) file
		if !utils.FileExists(configFile) {
			logger.Warn("Provider configuration file missing.")

			// Try to copy template if it exists
			templatePath := "templates/configuration.toml"
			if utils.FileExists(templatePath) {
				logger.Info("Copying configuration template to: " + logger.Highlight(configFile))
				if err := utils.CopyFile(templatePath, configFile); err != nil {
					return fmt.Errorf("failed to copy config template: %w", err)
				}
				logger.Info("Configuration template copied successfully!")
				logger.Warn("Please edit " + logger.Highlight(configFile) + " and add your provider API keys")
				return fmt.Errorf("configuration file created - please add your API keys and try again")
			}

			logger.Error("Configuration template not found at: " + templatePath)
			return fmt.Errorf("missing config file: %s", configFile)
		}
		logger.Info("Provider configuration file exists: " + logger.Highlight(configFile))

		// SSH keys will be created per-provider as needed
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
