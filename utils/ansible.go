package utils

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/emancipat3r/vps3/logger"
	"github.com/emancipat3r/vps3/ui"
)

var pathWg string

// CreateSSHWrapperForAnsible creates a wrapper script that Ansible can use for SSH with passphrase
func CreateSSHWrapperForAnsible(privKeyPath string) (string, error) {
	// Check if key has passphrase
	keyName := filepath.Base(privKeyPath)
	passFile := filepath.Join(filepath.Dir(filepath.Dir(privKeyPath)), "secrets", keyName+".pass")

	wrapperPath := fmt.Sprintf("/tmp/ansible_ssh_wrapper_%d.sh", os.Getpid())

	var wrapperContent string
	if passBytes, err := os.ReadFile(passFile); err == nil {
		// Key has passphrase, create wrapper with sshpass
		passphrase := strings.TrimSpace(string(passBytes))

		// Check if sshpass is available
		if _, err := exec.LookPath("sshpass"); err == nil {
			wrapperContent = fmt.Sprintf(`#!/bin/sh
exec sshpass -p '%s' ssh "$@"
`, passphrase)
		} else {
			// Use SSH_ASKPASS mechanism
			askpassScript := fmt.Sprintf("/tmp/ssh_askpass_%d.sh", os.Getpid())
			askpassContent := fmt.Sprintf("#!/bin/sh\necho '%s'\n", passphrase)
			if err := os.WriteFile(askpassScript, []byte(askpassContent), 0700); err != nil {
				return "", fmt.Errorf("failed to create askpass script: %w", err)
			}

			wrapperContent = fmt.Sprintf(`#!/bin/sh
export SSH_ASKPASS=%s
export SSH_ASKPASS_REQUIRE=force
exec ssh "$@"
`, askpassScript)
		}
	} else {
		// No passphrase, simple wrapper
		wrapperContent = `#!/bin/sh
exec ssh "$@"
`
	}

	if err := os.WriteFile(wrapperPath, []byte(wrapperContent), 0700); err != nil {
		return "", fmt.Errorf("failed to create SSH wrapper: %w", err)
	}

	return wrapperPath, nil
}

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
func GenerateInventory(config AnsibleConfig, pathAnsibleInventory string) error {
	// Read the inventory template
	templatePath := filepath.Join("templates", "inventory")
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return logger.Errorf("failed to read inventory template: %w", err)
	}

	// Parse and execute template
	tmpl, err := template.New("inventory").Parse(string(templateContent))
	if err != nil {
		return logger.Errorf("failed to parse inventory template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return fmt.Errorf("failed to execute inventory template: %w", err)
	}

	// Write inventory file
	if err := os.WriteFile(pathAnsibleInventory, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write inventory file: %w", err)
	}

	return nil
}

// WaitForSSH waits for SSH to become available on the VPS
func WaitForSSH(ip, privKeyPath string, maxWaitTime time.Duration) error {
	// Start the spinner
	ctx, cancel := context.WithTimeout(context.Background(), maxWaitTime)
	defer cancel()

	spinnerProg, doneChan := ui.IPWaitSpinner(ctx, "Waiting for SSH port to become available...")

	checkInterval := 10 * time.Second
	sshReady := false

	go func() {
		defer close(doneChan)

		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		// Check immediately, then continue with ticker
		for {
			// First check if port 22 is reachable (like netcat)
			if !IsPortReachable(ip, "22", 2*time.Second) {
				ui.UpdateSpinnerMessage(spinnerProg, fmt.Sprintf("Waiting for SSH port to become available..."))
			} else {
				// Port is open, now try actual SSH connection
				if !sshReady {
					ui.UpdateSpinnerMessage(spinnerProg, "Port 22 is open, testing SSH connection...")
					sshReady = true
				}

				// Test SSH connection
				// Check if SSH_AUTH_SOCK is set (ssh-agent is running)
				sshAuthSock := os.Getenv("SSH_AUTH_SOCK")
				useSystemSSH := false

				if sshAuthSock != "" && CheckKeyInSSHAgent(privKeyPath) {
					// ssh-agent is available and key is loaded
					useSystemSSH = true
				} else if sshAuthSock != "" {
					// ssh-agent is running but key not loaded
					logger.Info(fmt.Sprintf("SSH key not loaded in ssh-agent. To add it: ssh-add %s", privKeyPath))
					logger.Info("Falling back to Go SSH client with passphrase from secrets file")
				}

				if useSystemSSH {
					// Use system SSH with ssh-agent
					cmd := exec.Command("ssh",
						"-i", privKeyPath,
						"-o", "StrictHostKeyChecking=no",
						"-o", "UserKnownHostsFile=/dev/null",
						"-o", "ConnectTimeout=10",
						"-o", "IdentitiesOnly=yes",
						fmt.Sprintf("root@%s", ip),
						"echo 'SSH connection test'",
					)

					if err := cmd.Run(); err == nil {
						ui.FinishSpinner(spinnerProg, true, "")
						return
					} else {
						logger.Debug(fmt.Sprintf("SSH connection test failed: %v", err))
					}
				} else {
					// Use Go SSH client which handles passphrases
					logger.Debug("Using Go SSH client for authentication")
					sshClient, err := NewSSHClient(ip, "22", "root", privKeyPath)
					if err != nil {
						logger.Debug(fmt.Sprintf("Failed to create SSH client: %v", err))
					} else {
						if err := sshClient.Connect(); err == nil {
							// Test with a simple command
							if output, err := sshClient.ExecuteCommand("echo 'SSH connection test'"); err == nil {
								logger.Debug(fmt.Sprintf("SSH test output: %s", strings.TrimSpace(output)))
								ui.FinishSpinner(spinnerProg, true, "")
								sshClient.Close()
								return
							} else {
								logger.Debug(fmt.Sprintf("SSH command test failed: %v", err))
							}
							sshClient.Close()
						} else {
							logger.Debug(fmt.Sprintf("SSH connection failed: %v", err))
						}
					}
				}

				// Update spinner message to show we're still trying
				ui.UpdateSpinnerMessage(spinnerProg, "SSH port open but not ready yet...")
			}

			// Wait for next check interval or context done
			select {
			case <-ctx.Done():
				ui.FinishSpinner(spinnerProg, false, fmt.Sprintf("Timeout waiting for SSH after %v", maxWaitTime))
				return
			case <-ticker.C:
				// Continue to next iteration
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
		return nil
	}
}

// CheckOSCompatibility checks if the VPS is running Ubuntu or Debian
func CheckOSCompatibility(ip, privKeyPath string) (bool, string, error) {

	// Use Go SSH client to avoid passphrase prompts
	sshClient, err := NewSSHClient(ip, "22", "root", privKeyPath)
	if err != nil {
		return false, "", fmt.Errorf("failed to create SSH client: %w", err)
	}
	defer sshClient.Close()

	if err := sshClient.Connect(); err != nil {
		return false, "", fmt.Errorf("failed to connect: %w", err)
	}

	output, err := sshClient.ExecuteCommand("cat /etc/os-release | grep '^ID=' | cut -d= -f2 | tr -d '\"'")
	if err != nil {
		return false, "", fmt.Errorf("failed to check OS: %w", err)
	}

	osID := strings.TrimSpace(output)
	logger.Info("Detected OS: " + logger.Highlight(osID))

	compatible := osID == "ubuntu" || osID == "debian"
	return compatible, osID, nil
}

// RunAnsiblePlaybook executes the Ansible playbook
func RunAnsiblePlaybook(pathAnsibleInventory, playbookPath, privKeyPath string, verbose bool) error {
	logger.Info(strings.Repeat("=", 70))

	// Create SSH wrapper to handle passphrase
	wrapperPath, err := CreateSSHWrapperForAnsible(privKeyPath)
	if err != nil {
		logger.Warn(fmt.Sprintf("Could not create SSH wrapper: %v", err))
	} else {
		defer os.Remove(wrapperPath)
	}

	// Build ansible-playbook command arguments
	args := []string{
		"-i", pathAnsibleInventory,
		playbookPath,
		"--ssh-common-args=-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null",
	}

	// Check for verbose environment variable or parameter
	if verbose || os.Getenv("ANSIBLE_VERBOSE") != "" || os.Getenv("VPS3_DEBUG") != "" {
		args = append(args, "-v")
		logger.Info("Running in verbose mode...")
	}

	cmd := exec.Command("ansible-playbook", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin // Allow for any prompts if needed

	// Set environment to use our SSH wrapper if created
	if wrapperPath != "" {
		cmd.Env = append(os.Environ(), "ANSIBLE_SSH_EXECUTABLE="+wrapperPath)
	}

	// Start the command and wait for it to complete
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ansible playbook execution failed: %w", err)
	}

	logger.Info(strings.Repeat("=", 70))
	logger.Info("Ansible playbook complete")
	return nil
}

// SetupPostProvisioningAnsible orchestrates the complete Ansible setup process
func SetupPostProvisioningAnsible(ip, privKeyPath, vpsName string) error {

	// Check if Ansible is installed
	if err := CheckAnsibleInstalled(); err != nil {
		return fmt.Errorf("ansible is not installed. To install: pip install ansible or apt install ansible")
	}

	// Wait for SSH to become available
	if err := WaitForSSH(ip, privKeyPath, 5*time.Minute); err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}

	// Check OS compatibility
	compatible, osID, err := CheckOSCompatibility(ip, privKeyPath)
	if err != nil {
		logger.Warn("Could not determine OS compatibility, attempting setup anyway: " + err.Error())
		// Continue anyway as the playbook will fail gracefully if OS is not supported
	} else if !compatible {
		return fmt.Errorf("OS '%s' is not supported by Ansible playbook (only Ubuntu/Debian)", osID)
	}

	// Generate inventory file
	config := AnsibleConfig{
		IP:          ip,
		PrivKeyPath: privKeyPath,
		VPSName:     vpsName,
	}

	user, err := user.Current()
	if err != nil {
		logger.Error("Failed to get current user: " + err.Error())
		return err
	}
	pathConfig := user.HomeDir + "/.config/vps/"
	pathAnsibleInventory := pathConfig + "ansible/inventory"
	if err := GenerateInventory(config, pathAnsibleInventory); err != nil {
		return fmt.Errorf("failed to generate inventory: %w", err)
	}

	// Run the playbook (check for verbose mode)
	playbookPath := filepath.Join("ansible", "playbook.yml")
	verbose := os.Getenv("ANSIBLE_VERBOSE") != "" || os.Getenv("VPS3_DEBUG") != ""

	if err := RunAnsiblePlaybook(pathAnsibleInventory, playbookPath, privKeyPath, verbose); err != nil {
		return fmt.Errorf("failed to run playbook: %w", err)
	}

	// Download client configuration
	if err := DownloadClientConfig(ip, privKeyPath, vpsName); err != nil {
		logger.Warn("Failed to download WireGuard client configuration: " + err.Error())
		logger.Info("You can manually download it from: /root/wireguard-client/client.conf")
	}

	return nil
}

// DownloadClientConfig downloads the WireGuard client configuration from the VPS
func DownloadClientConfig(ip, privKeyPath, vpsName string) error {

	// Get current user
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	// Create local directory for client configs
	pathWg = currentUser.HomeDir + "/.config/vps/wg/"

	// Download client.conf
	scpCmd := exec.Command("scp",
		"-i", privKeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		fmt.Sprintf("root@%s:/root/wireguard-client/client.conf", ip),
		filepath.Join(pathWg, fmt.Sprintf("%s.conf", vpsName)),
	)

	if err := scpCmd.Run(); err != nil {
		return fmt.Errorf("failed to download client config: %w", err)
	}

	return nil
}
