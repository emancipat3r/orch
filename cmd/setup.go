package cmd

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
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
		// Check if instances file exists
		if _, err := os.Stat(instanceFile); os.IsNotExist(err) {
			logger.Error("No instances file found. Please create at least one VPS instance first.")
			return
		}

		// Load all instances
		var allInstances map[string]interface{}
		if _, err := toml.DecodeFile(instanceFile, &allInstances); err != nil {
			logger.Error("Failed to load instances file: " + err.Error())
			return
		}

		if len(allInstances) == 0 {
			logger.Error("No VPS instances found. Please create at least one VPS instance first.")
			return
		}

		// Create options for selection
		var options []string
		var instanceMap = make(map[string]map[string]interface{})

		for instanceID, instanceData := range allInstances {
			if instanceInfo, ok := instanceData.(map[string]interface{}); ok {
				provider := "Unknown"
				ip := "Unknown"
				label := instanceID

				if p, exists := instanceInfo["Provider"]; exists {
					if providerStr, ok := p.(string); ok {
						provider = providerStr
					}
				}
				if ipAddr, exists := instanceInfo["Ipv4"]; exists {
					if ipStr, ok := ipAddr.(string); ok {
						ip = ipStr
					}
				}
				if labelStr, exists := instanceInfo["Label"]; exists {
					if l, ok := labelStr.(string); ok {
						label = l
					}
				}

				displayText := fmt.Sprintf("%s - %s (%s) - %s", instanceID, label, provider, ip)
				options = append(options, displayText)
				instanceMap[displayText] = instanceInfo
			}
		}

		// Let user select instance
		selectedInstance := ui.ChoiceGeneric("Select VPS instance to setup:", options)
		if selectedInstance == "" {
			logger.Info("Operation cancelled by user.")
			return
		}

		// Get instance details
		instanceInfo := instanceMap[selectedInstance]

		// Extract required information
		var ip, privKeyPath, label string
		var ok bool

		if ipAddr, exists := instanceInfo["Ipv4"]; exists {
			if ip, ok = ipAddr.(string); !ok || ip == "" || ip == "pending" {
				logger.Error("Instance IP address is not available or pending.")
				return
			}
		} else {
			logger.Error("Instance IP address not found.")
			return
		}

		if keyPath, exists := instanceInfo["PrivKeyPath"]; exists {
			if privKeyPath, ok = keyPath.(string); !ok || privKeyPath == "" {
				logger.Error("Private key path not found.")
				return
			}
		} else {
			logger.Error("Private key path not found.")
			return
		}

		if labelStr, exists := instanceInfo["Label"]; exists {
			if label, ok = labelStr.(string); !ok {
				label = "instance"
			}
		} else {
			label = "instance"
		}

		// Use the label as default if vpsName is not provided via flag
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
