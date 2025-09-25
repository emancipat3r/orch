package cmd

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/emancipat3r/vps3/logger"
	"github.com/emancipat3r/vps3/ui"
	"github.com/emancipat3r/vps3/utils"
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
				label = "vps-instance"
			}
		} else {
			label = "vps-instance"
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
		logger.Info("  Private Key: " + logger.Highlight(privKeyPath))

		// Confirm with user
		confirm := ui.ChoiceGeneric("Proceed with post-provisioning setup?", []string{"Yes", "No"})
		if confirm != "Yes" {
			logger.Info("Operation cancelled by user.")
			return
		}

		// Run post-provisioning setup
		logger.Info("Starting post-provisioning setup...")
		if err := utils.SetupPostProvisioningGo(ip, privKeyPath, label); err != nil {
			logger.Error("Post-provisioning setup failed: " + err.Error())
			logger.Info("Please check the error message above and try again.")
			logger.Info("Common issues:")
			logger.Info("  - Instance might not be fully booted yet (wait a few minutes)")
			logger.Info("  - SSH key permissions might be incorrect (check key file permissions)")
			logger.Info("  - Firewall might be blocking SSH connections")
			return
		}

		logger.Info("Post-provisioning setup completed successfully!")
		logger.Info("Your VPS is now configured with:")
		logger.Info("  ✓ WireGuard VPN server")
		logger.Info("  ✓ Firewall (UFW) enabled")
		logger.Info("  ✓ SSH hardening applied")
		logger.Info("  ✓ Client configuration downloaded")

		logger.Info("Check the 'wireguard-clients' directory for your VPN configuration files.")
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
