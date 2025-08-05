package cmd

import (
	"fmt"
	"os"
	"os/user"

	"github.com/charmbracelet/keygen"
	"github.com/emancipat3r/vps3/logger"
	"github.com/emancipat3r/vps3/utils"
	"github.com/spf13/cobra"
)

var (
	pathConfig     string
	pathSSH        string
	pathSecrets    string
	pathInstances  string
	instanceFile   string
	privateKeyPath string
	publicKeyPath  string
	configFile     string
)

// rootCmd is the base command for the CLI
var rootCmd = &cobra.Command{
	Use:   "vps",
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

		pathConfig = user.HomeDir + "/.config/vps/config/"
		pathSSH = user.HomeDir + "/.config/vps/.ssh/"
		pathSecrets = user.HomeDir + "/.config/vps/secrets/"
		pathInstances = user.HomeDir + "/.config/vps/instances/"
		instanceFile = pathInstances + "instances.toml"
		privateKeyPath = pathSSH + "id_ed25519"
		publicKeyPath = pathSSH + "id_ed25519.pub"
		configFile = pathConfig + "configuration.toml"

		// Ensure all required directories exist
		for _, dir := range []string{pathConfig, pathSSH, pathSecrets, pathInstances} {
			if !utils.DirExists(dir) {
				logger.Warn(fmt.Sprintf("Directory doesn't exist. Creating: %s", logger.Highlight(dir)))
				if err := utils.MakeDirectory(dir); err != nil {
					logger.Error("Failed to create directory: " + err.Error())
					return err
				}
				logger.Success("Directory created: " + logger.Highlight(dir))
			} else {
				logger.Info("Directory exists: " + logger.Highlight(dir))
			}
		}

		// Check for credentials (config) file
		if !utils.FileExists(configFile) {
			logger.Warn("Provider configuration file missing. Exiting...")
			return fmt.Errorf("missing config file: %s", configFile)
		}
		logger.Info("Provider configuration file exists: " + logger.Highlight(configFile))

		// Check for SSH keys, and generate if missing
		if !(utils.FileExists(privateKeyPath) && utils.FileExists(publicKeyPath)) {
			logger.Warn("SSH keypair missing. Creating...")

			password, err := utils.GenerateRandomPassword(30)
			if err != nil {
				logger.Error("Failed to generate random password for SSH keypair: " + err.Error())
				return err
			}

			err = os.WriteFile(pathSecrets+"sshkeysecret.txt", []byte(password+"\n"), 0600)
			if err != nil {
				logger.Error("Failed to write SSH key password to file: " + logger.Highlight(pathSecrets+"sshkeysecret.txt"))
				return err
			}

			_, err = keygen.New(privateKeyPath, keygen.WithKeyType(keygen.Ed25519), keygen.WithPassphrase(password), keygen.WithWrite())
			if err != nil {
				logger.Error("Failed to create SSH keypair: " + err.Error())
				return err
			}

			if !(utils.FileExists(privateKeyPath) && utils.FileExists(publicKeyPath)) {
				logger.Error("Still failed to create SSH keypair (files still missing).")
				return fmt.Errorf("failed to create ssh keypair")
			}

			logger.Info("SSH private key exists: " + logger.Highlight(privateKeyPath))
			logger.Info("SSH public key exists: " + logger.Highlight(publicKeyPath))
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
