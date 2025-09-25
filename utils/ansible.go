package utils

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/emancipat3r/vps3/logger"
	"github.com/emancipat3r/vps3/ui"
)

type AnsibleConfig struct {
	IP          string
	PrivKeyPath string
	VPSName     string
}

// CheckAnsibleInstalled verifies if Ansible is installed on the system
func CheckAnsibleInstalled() error {
	cmd := exec.Command("ansible", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ansible is not installed or not in PATH. Please install ansible: %w", err)
	}
	return nil
}

// GenerateInventory creates an Ansible inventory file for the VPS
func GenerateInventory(config AnsibleConfig, inventoryPath string) error {
	// Read the inventory template
	templatePath := filepath.Join("ansible", "inventory.j2")
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read inventory template: %w", err)
	}

	// Parse and execute template
	tmpl, err := template.New("inventory").Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse inventory template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return fmt.Errorf("failed to execute inventory template: %w", err)
	}

	// Write inventory file
	if err := os.WriteFile(inventoryPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write inventory file: %w", err)
	}

	return nil
}

// WaitForSSH waits for SSH to become available on the VPS
func WaitForSSH(ip, privKeyPath string, maxWaitTime time.Duration) error {
	// Start the spinner
	ctx, cancel := context.WithTimeout(context.Background(), maxWaitTime)
	defer cancel()

	spinnerProg, doneChan := ui.IPWaitSpinner(ctx, "Waiting for SSH to become available...")

	checkInterval := 10 * time.Second

	go func() {
		defer close(doneChan)

		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				ui.FinishSpinner(spinnerProg, false, fmt.Sprintf("Timeout waiting for SSH after %v", maxWaitTime))
				return
			case <-ticker.C:
				// Test SSH connection
				cmd := exec.Command("ssh",
					"-i", privKeyPath,
					"-o", "StrictHostKeyChecking=no",
					"-o", "UserKnownHostsFile=/dev/null",
					"-o", "ConnectTimeout=10",
					"-o", "BatchMode=yes",
					fmt.Sprintf("root@%s", ip),
					"echo 'SSH connection test'",
				)

				if err := cmd.Run(); err == nil {
					ui.FinishSpinner(spinnerProg, true, "")
					return
				}
				// Update spinner message to show we're still trying
				ui.UpdateSpinnerMessage(spinnerProg, "Still waiting for SSH to become available...")
			}
		}
	}()

	// Wait for the spinner to complete
	<-doneChan

	// Add small delay to ensure spinner is fully cleared
	time.Sleep(100 * time.Millisecond)

	// Check if we succeeded or timed out
	select {
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for SSH to become available after %v", maxWaitTime)
	default:
		logger.Info("SSH connection established successfully")
		return nil
	}
}

// CheckOSCompatibility checks if the VPS is running Ubuntu or Debian
func CheckOSCompatibility(ip, privKeyPath string) (bool, string, error) {
	logger.Info("Checking OS compatibility...")

	cmd := exec.Command("ssh",
		"-i", privKeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		fmt.Sprintf("root@%s", ip),
		"cat /etc/os-release | grep '^ID=' | cut -d= -f2 | tr -d '\"'",
	)

	output, err := cmd.Output()
	if err != nil {
		return false, "", fmt.Errorf("failed to check OS: %w", err)
	}

	osID := strings.TrimSpace(string(output))
	logger.Info("Detected OS: " + logger.Highlight(osID))

	compatible := osID == "ubuntu" || osID == "debian"
	return compatible, osID, nil
}

// RunAnsiblePlaybook executes the Ansible playbook
func RunAnsiblePlaybook(inventoryPath, playbookPath string, verbose bool) error {
	logger.Info("Running Ansible playbook...")

	args := []string{
		"-i", inventoryPath,
		playbookPath,
		"--ssh-common-args=-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null",
	}

	if verbose {
		args = append(args, "-v")
	}

	cmd := exec.Command("ansible-playbook", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ansible playbook execution failed: %w", err)
	}

	logger.Info("Ansible playbook completed successfully")
	return nil
}

// SetupPostProvisioningAnsible orchestrates the complete Ansible setup process
func SetupPostProvisioningAnsible(ip, privKeyPath, vpsName string) error {
	// Check if Ansible is installed
	if err := CheckAnsibleInstalled(); err != nil {
		logger.Warn("Ansible is not installed, skipping post-provisioning setup")
		logger.Info("To install Ansible: pip install ansible")
		return nil
	}

	// Wait for SSH to become available
	if err := WaitForSSH(ip, privKeyPath, 5*time.Minute); err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}

	// Check OS compatibility
	compatible, osID, err := CheckOSCompatibility(ip, privKeyPath)
	if err != nil {
		logger.Warn("Could not determine OS compatibility, skipping Ansible setup: " + err.Error())
		return nil
	}

	if !compatible {
		logger.Warn(fmt.Sprintf("OS '%s' is not supported (only Ubuntu/Debian), skipping Ansible setup", osID))
		return nil
	}

	logger.Info("OS is compatible, proceeding with Ansible setup")

	// Generate inventory file
	config := AnsibleConfig{
		IP:          ip,
		PrivKeyPath: privKeyPath,
		VPSName:     vpsName,
	}

	inventoryPath := filepath.Join("ansible", "inventory")
	if err := GenerateInventory(config, inventoryPath); err != nil {
		return fmt.Errorf("failed to generate inventory: %w", err)
	}

	// Run the playbook
	playbookPath := filepath.Join("ansible", "playbook.yml")
	if err := RunAnsiblePlaybook(inventoryPath, playbookPath, false); err != nil {
		return fmt.Errorf("failed to run playbook: %w", err)
	}

	// Download client configuration
	if err := DownloadClientConfig(ip, privKeyPath); err != nil {
		logger.Warn("Failed to download WireGuard client configuration: " + err.Error())
	}

	return nil
}

// DownloadClientConfig downloads the WireGuard client configuration from the VPS
func DownloadClientConfig(ip, privKeyPath string) error {
	logger.Info("Downloading WireGuard client configuration...")

	// Create local directory for client configs
	clientDir := "wireguard-clients"
	if err := os.MkdirAll(clientDir, 0755); err != nil {
		return fmt.Errorf("failed to create client directory: %w", err)
	}

	// Download client.conf
	scpCmd := exec.Command("scp",
		"-i", privKeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		fmt.Sprintf("root@%s:/root/wireguard-client/client.conf", ip),
		filepath.Join(clientDir, fmt.Sprintf("client-%s.conf", ip)),
	)

	if err := scpCmd.Run(); err != nil {
		return fmt.Errorf("failed to download client config: %w", err)
	}

	logger.Info("WireGuard client configuration downloaded to: " + logger.Highlight(filepath.Join(clientDir, fmt.Sprintf("client-%s.conf", ip))))
	return nil
}

// SetupNetworkNamespace creates a network namespace and configures WireGuard client
func SetupNetworkNamespace(clientConfigPath, namespaceName string) error {
	if namespaceName == "" {
		namespaceName = "vps"
	}

	logger.Info("Setting up network namespace: " + logger.Highlight(namespaceName))

	// Check if running as root (required for network namespaces)
	if os.Geteuid() != 0 {
		logger.Warn("Network namespace setup requires root privileges")
		logger.Info("To set up the namespace manually, run as root:")
		logger.Info("  ip netns add " + namespaceName)
		logger.Info("  ip netns exec " + namespaceName + " wg-quick up " + clientConfigPath)
		return nil
	}

	// Create network namespace
	cmd := exec.Command("ip", "netns", "add", namespaceName)
	if err := cmd.Run(); err != nil {
		// Check if namespace already exists
		checkCmd := exec.Command("ip", "netns", "list")
		output, _ := checkCmd.Output()
		if strings.Contains(string(output), namespaceName) {
			logger.Info("Network namespace already exists: " + namespaceName)
		} else {
			return fmt.Errorf("failed to create network namespace: %w", err)
		}
	}

	// Set up WireGuard in the namespace
	wgCmd := exec.Command("ip", "netns", "exec", namespaceName, "wg-quick", "up", clientConfigPath)
	if err := wgCmd.Run(); err != nil {
		return fmt.Errorf("failed to setup WireGuard in namespace: %w", err)
	}

	logger.Info("Network namespace setup complete!")
	logger.Info("To use the namespace: ip netns exec " + namespaceName + " <command>")
	return nil
}
