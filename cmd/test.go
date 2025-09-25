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

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test SSH connectivity and system info for VPS instances",
	Long: `Test SSH connectivity and retrieve system information from existing VPS instances.
This is useful for troubleshooting connection issues and verifying instance status.`,
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
		selectedInstance := ui.ChoiceGeneric("Select VPS instance to test:", options)
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
			logger.Error("Please ensure the SSH key file exists before running test.")
			return
		}

		logger.Info("Testing VPS instance:")
		logger.Info("  IP Address: " + logger.Highlight(ip))
		logger.Info("  Label: " + logger.Highlight(label))
		logger.Info("  Private Key: " + logger.Highlight(privKeyPath))

		// Test SSH connectivity
		logger.Info("Testing SSH connectivity...")
		if err := utils.TestSSHConnectivity(ip, "22", "root", privKeyPath); err != nil {
			logger.Error("SSH connectivity test failed: " + err.Error())
			logger.Info("Troubleshooting tips:")
			logger.Info("  - Ensure the instance is fully booted (check cloud provider console)")
			logger.Info("  - Verify the IP address is correct")
			logger.Info("  - Check if your local firewall is blocking the connection")
			logger.Info("  - Verify the SSH key file has correct permissions (600)")
			return
		}

		// Create SSH client to get system info
		sshClient, err := utils.NewSSHClient(ip, "22", "root", privKeyPath)
		if err != nil {
			logger.Error("Failed to create SSH client: " + err.Error())
			return
		}

		if err := sshClient.Connect(); err != nil {
			logger.Error("Failed to connect to SSH: " + err.Error())
			return
		}
		defer sshClient.Close()

		// Get system information
		logger.Info("Retrieving system information...")
		systemInfo, err := utils.GetSystemInfo(sshClient)
		if err != nil {
			logger.Warn("Failed to get complete system info: " + err.Error())
		} else {
			logger.Info("System Information:")
			if hostname, exists := systemInfo["hostname"]; exists {
				logger.Info("  Hostname: " + logger.Highlight(hostname))
			}
			if osID, exists := systemInfo["os_id"]; exists {
				osVersion := systemInfo["os_version"]
				logger.Info("  OS: " + logger.Highlight(osID+" "+osVersion))
			}
			if kernel, exists := systemInfo["kernel"]; exists {
				logger.Info("  Kernel: " + logger.Highlight(kernel))
			}
			if uptime, exists := systemInfo["uptime"]; exists {
				logger.Info("  Uptime: " + logger.Highlight(uptime))
			}
			if wgStatus, exists := systemInfo["wireguard"]; exists {
				if wgStatus == "installed" {
					logger.Info("  WireGuard: " + logger.Highlight("✓ Installed"))
					if status, statusExists := systemInfo["wireguard_status"]; statusExists {
						if status == "active" {
							logger.Info("  WireGuard Status: " + logger.Highlight("✓ Running"))
						} else {
							logger.Info("  WireGuard Status: " + logger.Highlight("✗ Not Running ("+status+")"))
						}
					}
				} else {
					logger.Info("  WireGuard: " + logger.Highlight("✗ Not Installed"))
				}
			}
		}

		// Test specific ports
		logger.Info("Testing port connectivity...")

		// Test SSH port (should be open)
		if isOpen, err := utils.CheckPortOpen(sshClient, 22); err != nil {
			logger.Warn("Failed to check SSH port: " + err.Error())
		} else if isOpen {
			logger.Info("  SSH (22): " + logger.Highlight("✓ Open"))
		} else {
			logger.Info("  SSH (22): " + logger.Highlight("✗ Closed"))
		}

		// Test WireGuard port (51820)
		if isOpen, err := utils.CheckPortOpen(sshClient, 51820); err != nil {
			logger.Warn("Failed to check WireGuard port: " + err.Error())
		} else if isOpen {
			logger.Info("  WireGuard (51820): " + logger.Highlight("✓ Open"))
		} else {
			logger.Info("  WireGuard (51820): " + logger.Highlight("✗ Closed"))
		}

		// Check if WireGuard client config exists locally
		clientDir := "wireguard-clients"
		localConfigPath := fmt.Sprintf("%s/client-%s.conf", clientDir, ip)
		if _, err := os.Stat(localConfigPath); err == nil {
			logger.Info("  Client Config: " + logger.Highlight("✓ Available at "+localConfigPath))
		} else {
			logger.Info("  Client Config: " + logger.Highlight("✗ Not found locally"))
			logger.Info("    Run 'vps3 setup' to download the client configuration")
		}

		logger.Info("Test completed successfully!")
		logger.Info("Your VPS instance appears to be functioning correctly.")
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
}
