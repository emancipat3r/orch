package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/emancipat3r/vps3/logger"
	"github.com/spf13/cobra"
)

var namespaceCmd = &cobra.Command{
	Use:   "namespace",
	Short: "Manage network namespace for VPS connection",
	Long: `Manage network namespace for secure VPS connection using WireGuard.
This command helps you create, manage, and use network namespaces to route
traffic through your VPS connection.`,
}

var namespaceSetupCmd = &cobra.Command{
	Use:   "setup [namespace-name] [config-path]",
	Short: "Set up a new network namespace with WireGuard",
	Long: `Create a network namespace and configure WireGuard client connection.
All traffic in the namespace will be routed through the VPN.

Examples:
  vps3 namespace setup
  vps3 namespace setup myvps
  vps3 namespace setup myvps /path/to/client.conf`,
	Args: cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		namespaceName := "vps"
		configPath := ""

		if len(args) > 0 {
			namespaceName = args[0]
		}
		if len(args) > 1 {
			configPath = args[1]
		}

		// Check if running as root
		if os.Geteuid() != 0 {
			logger.Error("This command must be run as root")
			logger.Info("Please run: sudo vps3 namespace setup " + namespaceName)
			return
		}

		scriptPath := filepath.Join("scripts", "setup-namespace.sh")
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			logger.Error("Setup script not found: " + scriptPath)
			logger.Info("Make sure you're running this from the vps3 directory")
			return
		}

		var setupCmd *exec.Cmd
		if configPath != "" {
			setupCmd = exec.Command(scriptPath, namespaceName, configPath)
		} else {
			setupCmd = exec.Command(scriptPath, namespaceName)
		}

		setupCmd.Stdout = os.Stdout
		setupCmd.Stderr = os.Stderr
		setupCmd.Stdin = os.Stdin

		if err := setupCmd.Run(); err != nil {
			logger.Error("Failed to set up namespace: " + err.Error())
			return
		}
	},
}

var namespaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all network namespaces",
	Long:  "List all available network namespaces on the system.",
	Run: func(cmd *cobra.Command, args []string) {
		listCmd := exec.Command("ip", "netns", "list")
		output, err := listCmd.Output()
		if err != nil {
			logger.Error("Failed to list namespaces: " + err.Error())
			return
		}

		if len(output) == 0 {
			logger.Info("No network namespaces found")
			return
		}

		logger.Info("Available network namespaces:")
		fmt.Print(string(output))
	},
}

var namespaceExecCmd = &cobra.Command{
	Use:   "exec [namespace-name] [command...]",
	Short: "Execute a command in a network namespace",
	Long: `Execute a command inside a network namespace.
All network traffic from the command will go through the VPS connection.

Examples:
  vps3 namespace exec vps curl ipinfo.io
  vps3 namespace exec vps bash
  vps3 namespace exec vps ping 8.8.8.8`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		namespaceName := args[0]
		command := args[1:]

		// Check if running as root
		if os.Geteuid() != 0 {
			logger.Error("This command must be run as root")
			logger.Info("Please run: sudo vps3 namespace exec " + namespaceName + " " + fmt.Sprintf("%v", command))
			return
		}

		// Build the ip netns exec command
		execArgs := []string{"netns", "exec", namespaceName}
		execArgs = append(execArgs, command...)

		execCmd := exec.Command("ip", execArgs...)
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		execCmd.Stdin = os.Stdin

		if err := execCmd.Run(); err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				os.Exit(exitError.ExitCode())
			}
			logger.Error("Failed to execute command: " + err.Error())
			return
		}
	},
}

var namespaceDeleteCmd = &cobra.Command{
	Use:   "delete [namespace-name]",
	Short: "Delete a network namespace",
	Long: `Delete a network namespace and clean up associated resources.
This will stop any WireGuard connections and remove the namespace.

Examples:
  vps3 namespace delete vps
  vps3 namespace delete myvps`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		namespaceName := args[0]

		// Check if running as root
		if os.Geteuid() != 0 {
			logger.Error("This command must be run as root")
			logger.Info("Please run: sudo vps3 namespace delete " + namespaceName)
			return
		}

		// Check if namespace exists
		listCmd := exec.Command("ip", "netns", "list")
		output, err := listCmd.Output()
		if err != nil {
			logger.Error("Failed to check namespaces: " + err.Error())
			return
		}

		namespaceExists := false
		for _, line := range []string{string(output)} {
			if line == namespaceName || line == namespaceName+" (id: " {
				namespaceExists = true
				break
			}
		}

		if !namespaceExists {
			logger.Warn("Namespace '" + namespaceName + "' does not exist")
			return
		}

		logger.Info("Deleting namespace: " + namespaceName)

		// Try to bring down any WireGuard interfaces first
		wgDownCmd := exec.Command("ip", "netns", "exec", namespaceName, "wg-quick", "down", "all")
		wgDownCmd.Run() // Ignore errors, interface might not exist

		// Delete the namespace
		deleteCmd := exec.Command("ip", "netns", "delete", namespaceName)
		if err := deleteCmd.Run(); err != nil {
			logger.Error("Failed to delete namespace: " + err.Error())
			return
		}

		logger.Info("Namespace '" + namespaceName + "' deleted successfully")
	},
}

var namespaceStatusCmd = &cobra.Command{
	Use:   "status [namespace-name]",
	Short: "Show status of a network namespace",
	Long: `Show detailed status information about a network namespace,
including WireGuard connection status and network configuration.

Examples:
  vps3 namespace status vps
  vps3 namespace status myvps`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		namespaceName := args[0]

		// Check if namespace exists
		listCmd := exec.Command("ip", "netns", "list")
		output, err := listCmd.Output()
		if err != nil {
			logger.Error("Failed to check namespaces: " + err.Error())
			return
		}

		namespaceExists := false
		for _, line := range []string{string(output)} {
			if line == namespaceName || line == namespaceName+" (id: " {
				namespaceExists = true
				break
			}
		}

		if !namespaceExists {
			logger.Error("Namespace '" + namespaceName + "' does not exist")
			return
		}

		logger.Info("Status for namespace: " + namespaceName)
		fmt.Println()

		// Show WireGuard status
		logger.Info("WireGuard status:")
		wgCmd := exec.Command("ip", "netns", "exec", namespaceName, "wg", "show")
		wgCmd.Stdout = os.Stdout
		wgCmd.Stderr = os.Stderr
		if err := wgCmd.Run(); err != nil {
			logger.Warn("No WireGuard interfaces found or WireGuard not running")
		}

		fmt.Println()

		// Show network interfaces
		logger.Info("Network interfaces:")
		ipCmd := exec.Command("ip", "netns", "exec", namespaceName, "ip", "addr", "show")
		ipCmd.Stdout = os.Stdout
		ipCmd.Stderr = os.Stderr
		ipCmd.Run()

		fmt.Println()

		// Test connectivity
		logger.Info("Testing connectivity...")
		pingCmd := exec.Command("ip", "netns", "exec", namespaceName, "ping", "-c", "1", "-W", "5", "8.8.8.8")
		if err := pingCmd.Run(); err != nil {
			logger.Warn("Connectivity test failed")
		} else {
			logger.Info("Connectivity test passed")
		}
	},
}

func init() {
	rootCmd.AddCommand(namespaceCmd)

	namespaceCmd.AddCommand(namespaceSetupCmd)
	namespaceCmd.AddCommand(namespaceListCmd)
	namespaceCmd.AddCommand(namespaceExecCmd)
	namespaceCmd.AddCommand(namespaceDeleteCmd)
	namespaceCmd.AddCommand(namespaceStatusCmd)
}
