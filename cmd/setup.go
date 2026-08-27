package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/emancipat3r/orch/logger"
	"github.com/emancipat3r/orch/ui"
	"github.com/emancipat3r/orch/utils"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Run post-provisioning setup on existing VPS instances",
	Long: `Run post-provisioning setup (WireGuard, firewall, SSH hardening) on existing VPS instances.
This is useful if the initial setup failed or if you want to reconfigure an instance.`,
	Run: func(cmd *cobra.Command, args []string) {
		db, err := utils.LoadInstances(instanceFile)
		if err != nil {
			logger.Error("Failed to load instances file: " + err.Error())
			return
		}
		if len(db) == 0 {
			logger.Error("No VPS instances found. Please create at least one VPS instance first.")
			return
		}

		// Stable ordering so the menu doesn't reshuffle between runs.
		ids := make([]string, 0, len(db))
		for id := range db {
			ids = append(ids, id)
		}
		sort.Strings(ids)

		options := make([]string, 0, len(db))
		byOption := make(map[string]utils.InstanceRecord, len(db))
		for _, id := range ids {
			rec := db[id]
			provider, ip, label := rec.Provider, rec.Ipv4, rec.Label
			if provider == "" {
				provider = "Unknown"
			}
			if ip == "" {
				ip = "Unknown"
			}
			if label == "" {
				label = id
			}
			displayText := fmt.Sprintf("%s - %s (%s) - %s", id, label, provider, ip)
			options = append(options, displayText)
			byOption[displayText] = rec
		}

		selectedInstance := ui.ChoiceGeneric("Select VPS instance to setup:", options)
		if selectedInstance == "" {
			logger.Info("Operation cancelled by user.")
			return
		}
		rec := byOption[selectedInstance]

		ip := rec.Ipv4
		if ip == "" || ip == "pending" {
			logger.Error("Instance IP address is not available or pending.")
			return
		}
		privKeyPath := rec.PrivKeyPath
		if privKeyPath == "" {
			logger.Error("Private key path not found.")
			return
		}
		label := rec.Label
		if label == "" {
			label = "instance"
		}

		// Prefer the name the instance was created with so setup writes the
		// client config to the same file create/destroy use; fall back to the
		// provider label only for pre-name registry entries.
		if vpsName == "" {
			vpsName = rec.VPSName
		}
		if vpsName == "" {
			vpsName = label
		}

		// Check if private key file exists
		if _, err := os.Stat(privKeyPath); os.IsNotExist(err) {
			logger.Error("Private key file not found: " + privKeyPath)
			logger.Error("Please ensure the SSH key file exists before running setup.")
			return
		}

		logger.Info("Setting up VPS instance:")
		logger.Info("  IP Address: " + logger.Highlight(ip))
		logger.Info("  Label: " + logger.Highlight(label))
		logger.Info("  VPS Name: " + logger.Highlight(vpsName))
		logger.Info("  Private Key: " + logger.Highlight(privKeyPath))

		// Confirm with user
		confirm := ui.ChoiceGeneric("Proceed with post-provisioning setup?", []string{"Yes", "No"})
		if confirm != "Yes" {
			logger.Info("Operation cancelled by user.")
			return
		}

		// Run post-provisioning setup
		if err := utils.SetupPostProvisioningAnsible(cmd.Context(), ip, privKeyPath, vpsName); err != nil {
			logger.Error("Post-provisioning setup failed: " + err.Error())
			return
		} else {
			logger.Info("Post-provisioning setup completed")
		}

	},
}

func init() {
	setupCmd.Flags().StringVarP(&vpsName, "name", "n", "", "Custom name for the VPS instance (defaults to instance label)")

	rootCmd.AddCommand(setupCmd)
}
