package utils

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emancipat3r/vps3/logger"
	"github.com/emancipat3r/vps3/ui"
)

type PostProvisionConfig struct {
	IP          string
	PrivKeyPath string
	VPSName     string
	WGPort      int
}

type WireGuardKeys struct {
	ServerPrivate string
	ServerPublic  string
	ClientPrivate string
	ClientPublic  string
}

// SetupPostProvisioningGo orchestrates the complete post-provisioning setup in pure Go
func SetupPostProvisioningGo(ip, privKeyPath, vpsName string) error {
	config := PostProvisionConfig{
		IP:          ip,
		PrivKeyPath: privKeyPath,
		VPSName:     vpsName,
		WGPort:      51820,
	}

	// Create SSH client and wait for connection
	sshClient, err := NewSSHClient(config.IP, "22", "root", config.PrivKeyPath)
	if err != nil {
		return fmt.Errorf("failed to create SSH client: %w", err)
	}

	// Wait for SSH to become available with retry logic
	if err := waitForSSHConnectionWithRetry(sshClient, 10*time.Minute); err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}
	defer sshClient.Close()

	logger.Info("SSH connection established successfully")

	// Get system info and check OS compatibility
	systemInfo, err := GetSystemInfo(sshClient)
	if err != nil {
		return fmt.Errorf("failed to get system info: %w", err)
	}

	osID := systemInfo["os_id"]
	if osID != "ubuntu" && osID != "debian" {
		return fmt.Errorf("unsupported OS '%s' (only Ubuntu/Debian supported)", osID)
	}

	logger.Info("OS is compatible: " + logger.Highlight(osID+" "+systemInfo["os_version"]))

	// Run post-provisioning setup
	if err := runPostProvisioningSetup(sshClient, config); err != nil {
		return fmt.Errorf("post-provisioning setup failed: %w", err)
	}

	// Download client configuration
	if err := downloadClientConfig(sshClient, config); err != nil {
		logger.Warn("Failed to download WireGuard client configuration: " + err.Error())
	}

	logger.Info("Post-provisioning setup completed successfully")
	return nil
}

// waitForSSHConnectionWithRetry waits for SSH to become available with improved retry logic
func waitForSSHConnectionWithRetry(client *SSHClient, maxWaitTime time.Duration) error {
	// Start the spinner
	ctx, cancel := context.WithTimeout(context.Background(), maxWaitTime)
	defer cancel()

	spinnerProg, doneChan := ui.IPWaitSpinner(ctx, "Waiting for SSH port to become available...")

	connected := false
	checkInterval := 10 * time.Second
	sshPortOpen := false

	go func() {
		defer close(doneChan)

		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		// Check immediately, then continue with ticker
		for {
			// First check if port 22 is reachable (like netcat)
			logger.Debug(fmt.Sprintf("Checking if port 22 is reachable at %s:%s", client.Host, client.Port))
			if !IsPortReachable(client.Host, client.Port, 2*time.Second) {
				logger.Debug(fmt.Sprintf("Port 22 not reachable yet at %s:%s", client.Host, client.Port))
				ui.UpdateSpinnerMessage(spinnerProg, fmt.Sprintf("Waiting for port 22 to open at %s...", client.Host))
			} else {
				logger.Debug(fmt.Sprintf("Port 22 is now reachable at %s:%s", client.Host, client.Port))

				// Port is open, now try actual SSH connection
				if !sshPortOpen {
					logger.Info(fmt.Sprintf("Port 22 is open at %s, now testing SSH connection...", client.Host))
					ui.UpdateSpinnerMessage(spinnerProg, "Port 22 is open, testing SSH connection...")
					sshPortOpen = true
				}

				// Try to establish SSH connection
				if err := client.Connect(); err == nil {
					// Test with a simple command to ensure it's really working
					if _, err := client.ExecuteCommand("echo 'test'"); err == nil {
						connected = true
						logger.Info("SSH connection test successful")
						ui.FinishSpinner(spinnerProg, true, "")
						return
					} else {
						logger.Debug(fmt.Sprintf("SSH command test failed: %v", err))
					}
					client.Close() // Close the connection to retry
				} else {
					logger.Debug(fmt.Sprintf("SSH connection failed: %v", err))
				}
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
	if !connected {
		return fmt.Errorf("timeout waiting for SSH to become available after %v", maxWaitTime)
	}

	return nil
}

// generateWireGuardKey generates a WireGuard private key
func generateWireGuardKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// runPostProvisioningSetup performs all the setup tasks
func runPostProvisioningSetup(client *SSHClient, config PostProvisionConfig) error {
	logger.Info("Starting post-provisioning setup...")

	// Update system packages with diagnostics
	logger.Info("Updating system packages...")
	if err := updatePackagesWithDiagnostics(client); err != nil {
		logger.Warn("Package update failed. You can try manual recovery with these commands:")
		logger.Warn("ssh root@" + config.IP)
		logger.Warn("apt-get clean && rm -rf /var/lib/apt/lists/* && apt-get update")
		return fmt.Errorf("failed to update packages: %w", err)
	}

	// Wait for apt locks to be released
	logger.Info("Waiting for package manager locks...")
	waitCmd := `
while fuser /var/lib/dpkg/lock >/dev/null 2>&1 || \
    fuser /var/lib/apt/lists/lock >/dev/null 2>&1 || \
    fuser /var/cache/apt/archives/lock >/dev/null 2>&1; do
  echo "Waiting for other apt operations to complete..."
  sleep 5
done`
	if _, err := client.ExecuteCommand(waitCmd); err != nil {
		logger.Warn("Warning: could not wait for apt locks: " + err.Error())
	}

	// Upgrade packages
	logger.Info("Upgrading system packages...")
	if _, err := client.ExecuteCommand("DEBIAN_FRONTEND=noninteractive apt-get upgrade -y"); err != nil {
		return fmt.Errorf("failed to upgrade packages: %w", err)
	}

	// Install essential packages in groups to better identify issues
	logger.Info("Installing essential packages...")

	// First group: basic tools
	basicPackages := []string{"curl", "wget", "git", "netcat-openbsd", "socat"}
	if err := installPackageGroup(client, basicPackages, "basic tools"); err != nil {
		return err
	}

	// Second group: networking tools
	netPackages := []string{"net-tools", "ufw"}
	if err := installPackageGroup(client, netPackages, "networking tools"); err != nil {
		return err
	}

	// Third group: iptables-persistent (may require interaction)
	logger.Info("Installing iptables-persistent with pre-configuration...")
	// Pre-configure iptables-persistent to avoid interactive prompts
	preConfigCommands := []string{
		"echo iptables-persistent iptables-persistent/autosave_v4 boolean true | debconf-set-selections",
		"echo iptables-persistent iptables-persistent/autosave_v6 boolean true | debconf-set-selections",
	}
	for _, cmd := range preConfigCommands {
		if _, err := client.ExecuteCommand(cmd); err != nil {
			logger.Warn("Pre-configuration command failed: " + err.Error())
		}
	}
	iptablesPackages := []string{"iptables-persistent"}
	if err := installPackageGroup(client, iptablesPackages, "iptables tools"); err != nil {
		logger.Warn("Failed to install iptables-persistent, continuing without it: " + err.Error())
	}

	// Fourth group: qrencode
	qrPackages := []string{"qrencode"}
	if err := installPackageGroup(client, qrPackages, "QR code tools"); err != nil {
		logger.Warn("Failed to install qrencode, continuing without it: " + err.Error())
	}

	// Fifth group: WireGuard (ensure repository is available)
	if err := setupWireGuardRepository(client); err != nil {
		logger.Warn("Failed to setup WireGuard repository: " + err.Error())
	}

	wgPackages := []string{"wireguard", "wireguard-tools"}
	if err := installPackageGroup(client, wgPackages, "WireGuard"); err != nil {
		// Try alternative installation method
		logger.Warn("Standard WireGuard installation failed, trying alternative method...")
		if altErr := installWireGuardAlternative(client); altErr != nil {
			return fmt.Errorf("both standard and alternative WireGuard installation failed: %w (alt: %w)", err, altErr)
		}
	}

	// Enable IP forwarding
	logger.Info("Configuring IP forwarding...")
	if _, err := client.ExecuteCommand("echo 'net.ipv4.ip_forward=1' >> /etc/sysctl.conf"); err != nil {
		return fmt.Errorf("failed to enable IPv4 forwarding: %w", err)
	}
	if _, err := client.ExecuteCommand("echo 'net.ipv6.conf.all.forwarding=1' >> /etc/sysctl.conf"); err != nil {
		return fmt.Errorf("failed to enable IPv6 forwarding: %w", err)
	}
	if _, err := client.ExecuteCommand("sysctl -p"); err != nil {
		return fmt.Errorf("failed to apply sysctl changes: %w", err)
	}

	// Generate WireGuard keys
	logger.Info("Generating WireGuard keys...")
	wgKeys, err := generateWireGuardKeys(client)
	if err != nil {
		return fmt.Errorf("failed to generate WireGuard keys: %w", err)
	}

	// Create WireGuard configuration directory
	if err := CreateRemoteDirectory(client, "/etc/wireguard", "700"); err != nil {
		return fmt.Errorf("failed to create WireGuard directory: %w", err)
	}

	// Create WireGuard server configuration
	logger.Info("Creating WireGuard server configuration...")
	serverConfig := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = 10.0.0.1/24
ListenPort = %d
SaveConfig = true

# Enable packet forwarding
PostUp = iptables -A FORWARD -i %%i -j ACCEPT; iptables -A FORWARD -o %%i -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i %%i -j ACCEPT; iptables -D FORWARD -o %%i -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE

[Peer]
PublicKey = %s
AllowedIPs = 10.0.0.2/32
`, wgKeys.ServerPrivate, config.WGPort, wgKeys.ClientPublic)

	if err := CreateRemoteFile(client, "/etc/wireguard/wg0.conf", serverConfig, "600"); err != nil {
		return fmt.Errorf("failed to create server config: %w", err)
	}

	// Create client configuration directory
	if err := CreateRemoteDirectory(client, "/root/wireguard-client", "700"); err != nil {
		return fmt.Errorf("failed to create client config directory: %w", err)
	}

	// Create WireGuard client configuration
	logger.Info("Creating WireGuard client configuration...")
	clientConfig := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = 10.0.0.2/32
DNS = 1.1.1.1, 8.8.8.8

[Peer]
PublicKey = %s
Endpoint = %s:%d
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25
`, wgKeys.ClientPrivate, wgKeys.ServerPublic, config.IP, config.WGPort)

	if err := CreateRemoteFile(client, "/root/wireguard-client/client.conf", clientConfig, "600"); err != nil {
		return fmt.Errorf("failed to create client config: %w", err)
	}

	// Generate QR code for client configuration
	logger.Info("Generating QR code for client configuration...")
	if _, err := client.ExecuteCommand("qrencode -t ansiutf8 < /root/wireguard-client/client.conf > /root/wireguard-client/client-qr.txt"); err != nil {
		logger.Warn("Failed to generate QR code: " + err.Error())
	}

	// Configure UFW firewall
	logger.Info("Configuring firewall...")
	firewallCommands := []string{
		"ufw --force reset",
		fmt.Sprintf("ufw allow %d/udp", config.WGPort),
		"ufw allow 22/tcp",
		"ufw --force enable",
	}
	for _, cmd := range firewallCommands {
		if _, err := client.ExecuteCommand(cmd); err != nil {
			return fmt.Errorf("failed to configure firewall: %w", err)
		}
	}

	// Configure SSH security
	logger.Info("Configuring SSH security...")
	sshSecurityCommands := []string{
		"sed -i 's/^.PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config",
		"sed -i 's/^.AllowTcpForwarding.*/AllowTcpForwarding yes/' /etc/ssh/sshd_config",
		"sed -i 's/^.GatewayPorts.*/GatewayPorts yes/' /etc/ssh/sshd_config",
		"sed -i 's/^.PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config",
		"sed -i 's/^.PermitRootLogin.*/PermitRootLogin prohibit-password/' /etc/ssh/sshd_config",
	}
	for _, cmd := range sshSecurityCommands {
		if _, err := client.ExecuteCommand(cmd); err != nil {
			logger.Warn("SSH security config warning: " + err.Error())
		}
	}

	// Restart SSH service
	if _, err := client.ExecuteCommand("systemctl restart ssh"); err != nil {
		logger.Warn("Failed to restart SSH service: " + err.Error())
	}

	// Start and enable WireGuard
	logger.Info("Starting WireGuard service...")
	if _, err := client.ExecuteCommand("systemctl enable wg-quick@wg0"); err != nil {
		return fmt.Errorf("failed to enable WireGuard: %w", err)
	}
	if _, err := client.ExecuteCommand("systemctl start wg-quick@wg0"); err != nil {
		return fmt.Errorf("failed to start WireGuard: %w", err)
	}

	// Verify WireGuard is running
	if output, err := client.ExecuteCommand("systemctl is-active wg-quick@wg0"); err != nil {
		return fmt.Errorf("WireGuard failed to start: %w", err)
	} else if !strings.Contains(output, "active") {
		return fmt.Errorf("WireGuard is not active: %s", output)
	}

	logger.Info("WireGuard Server Public Key: " + logger.Highlight(strings.TrimSpace(wgKeys.ServerPublic)))
	logger.Info("Client configuration saved to /root/wireguard-client/client.conf")

	return nil
}

// installPackageGroup installs a group of packages with better error reporting
func installPackageGroup(client *SSHClient, packages []string, groupName string) error {
	logger.Info("Installing " + groupName + "...")

	// Try to fix common package issues first
	if err := fixPackageIssues(client); err != nil {
		logger.Warn("Package fix attempts failed: " + err.Error())
	}

	// Update package cache before installation
	logger.Info("Updating package cache...")
	if updateOutput, err := client.ExecuteCommand("apt-get update -q"); err != nil {
		logger.Warn("Warning: failed to update package cache: " + err.Error())
		logger.Warn("Update output: " + updateOutput)
	} else {
		logger.Info("Package cache updated successfully")
	}

	// Show available packages for debugging
	logger.Info("Checking package availability...")
	for _, pkg := range packages {
		if availOutput, err := client.ExecuteCommand(fmt.Sprintf("apt-cache policy %s", pkg)); err != nil {
			logger.Warn(fmt.Sprintf("Cannot find package %s in cache", pkg))
		} else {
			logger.Info(fmt.Sprintf("Package %s policy: %s", pkg, strings.Split(availOutput, "\n")[0]))
		}
	}

	installCmd := fmt.Sprintf("DEBIAN_FRONTEND=noninteractive apt-get install -y %s", strings.Join(packages, " "))
	logger.Info("Executing: " + installCmd)
	output, err := client.ExecuteCommand(installCmd)
	if err != nil {
		// Log the full error output for debugging
		logger.Warn("Package installation command failed: " + installCmd)
		logger.Warn("Error output: " + output)
		logger.Warn("Exit code indicates: " + err.Error())

		// Try to get more detailed error information
		logger.Warn("Package installation failed, checking individual packages...")

		// Try installing each package individually to identify the problematic one
		failedPackages := []string{}
		for _, pkg := range packages {
			// Check if package exists first
			checkCmd := fmt.Sprintf("apt-cache show %s > /dev/null 2>&1", pkg)
			if _, checkErr := client.ExecuteCommand(checkCmd); checkErr != nil {
				logger.Warn(fmt.Sprintf("Package %s not found in repositories", pkg))
				// Try alternative package name
				if altPkg := getPackageAlternative(pkg); altPkg != "" {
					logger.Info(fmt.Sprintf("Trying alternative package: %s", altPkg))
					altCmd := fmt.Sprintf("DEBIAN_FRONTEND=noninteractive apt-get install -y %s", altPkg)
					if _, altErr := client.ExecuteCommand(altCmd); altErr != nil {
						logger.Warn(fmt.Sprintf("Alternative package %s also failed", altPkg))
						failedPackages = append(failedPackages, pkg)
					} else {
						logger.Info(fmt.Sprintf("Successfully installed alternative %s", altPkg))
					}
				} else {
					failedPackages = append(failedPackages, pkg)
				}
				continue
			}

			singleCmd := fmt.Sprintf("DEBIAN_FRONTEND=noninteractive apt-get install -y %s", pkg)
			if singleOutput, singleErr := client.ExecuteCommand(singleCmd); singleErr != nil {
				logger.Warn(fmt.Sprintf("Failed to install %s: %s", pkg, singleErr.Error()))
				if singleOutput != "" {
					logger.Warn("Package output: " + singleOutput)
				}
				failedPackages = append(failedPackages, pkg)
			} else {
				logger.Info(fmt.Sprintf("Successfully installed %s", pkg))
			}
		}

		if len(failedPackages) == len(packages) {
			return fmt.Errorf("failed to install %s packages: %s (output: %s)", groupName, strings.Join(failedPackages, ", "), output)
		} else if len(failedPackages) > 0 {
			logger.Warn(fmt.Sprintf("Some %s packages failed to install: %s", groupName, strings.Join(failedPackages, ", ")))
		}
	}

	return nil
}

// getPackageAlternative returns alternative package names for common packages
func getPackageAlternative(pkg string) string {
	alternatives := map[string]string{
		"netcat-openbsd": "netcat",
		"net-tools":      "iproute2",
	}
	return alternatives[pkg]
}

// setupWireGuardRepository ensures WireGuard repository is available
func setupWireGuardRepository(client *SSHClient) error {
	logger.Info("Setting up WireGuard repository...")

	// Check if WireGuard packages are available
	if _, err := client.ExecuteCommand("apt-cache show wireguard > /dev/null 2>&1"); err == nil {
		logger.Info("WireGuard packages already available")
		return nil
	}

	// For Debian 12, WireGuard should be in main repository
	// Update sources.list to ensure all components are enabled
	sourcesCommands := []string{
		"echo 'deb http://deb.debian.org/debian bookworm main contrib non-free-firmware' > /etc/apt/sources.list.d/wireguard.list",
		"apt-get update",
	}

	for _, cmd := range sourcesCommands {
		if _, err := client.ExecuteCommand(cmd); err != nil {
			return fmt.Errorf("failed to setup repository: %w", err)
		}
	}

	return nil
}

// installWireGuardAlternative tries alternative WireGuard installation methods
func installWireGuardAlternative(client *SSHClient) error {
	logger.Info("Attempting alternative WireGuard installation...")

	// Try installing from backports
	backportCommands := []string{
		"echo 'deb http://deb.debian.org/debian bookworm-backports main' >> /etc/apt/sources.list",
		"apt-get update",
		"DEBIAN_FRONTEND=noninteractive apt-get install -y -t bookworm-backports wireguard wireguard-tools",
	}

	for i, cmd := range backportCommands {
		if _, err := client.ExecuteCommand(cmd); err != nil {
			logger.Warn(fmt.Sprintf("Backport command %d failed: %s", i+1, err.Error()))
			if i == 2 { // If the actual install failed, try manual kernel module approach
				return tryKernelModuleInstall(client)
			}
		}
	}

	return nil
}

// tryKernelModuleInstall attempts to install WireGuard kernel module manually
func tryKernelModuleInstall(client *SSHClient) error {
	logger.Info("Attempting kernel module installation...")

	moduleCommands := []string{
		"apt-get install -y linux-headers-$(uname -r) build-essential",
		"modprobe wireguard",
	}

	for _, cmd := range moduleCommands {
		if _, err := client.ExecuteCommand(cmd); err != nil {
			return fmt.Errorf("kernel module installation failed: %w", err)
		}
	}

	// Check if wireguard module is loaded
	if _, err := client.ExecuteCommand("lsmod | grep wireguard"); err != nil {
		return fmt.Errorf("WireGuard kernel module not loaded")
	}

	logger.Info("WireGuard kernel module successfully loaded")
	return nil
}

// updatePackagesWithDiagnostics updates packages with comprehensive diagnostics
func updatePackagesWithDiagnostics(client *SSHClient) error {
	// First, run diagnostics to understand the system state
	logger.Info("Running system diagnostics...")

	diagnostics := []struct {
		name string
		cmd  string
	}{
		{"Check DNS resolution", "nslookup deb.debian.org || dig deb.debian.org || ping -c 1 8.8.8.8"},
		{"Check internet connectivity", "curl -s --connect-timeout 10 http://deb.debian.org || wget -q --spider --timeout=10 http://deb.debian.org"},
		{"Check current sources", "cat /etc/apt/sources.list"},
		{"Check sources.list.d", "ls -la /etc/apt/sources.list.d/ && find /etc/apt/sources.list.d/ -name '*.list' -exec cat {} \\;"},
		{"Check disk space", "df -h"},
		{"Check existing locks", "ls -la /var/lib/apt/lists/lock* /var/cache/apt/archives/lock* /var/lib/dpkg/lock* 2>/dev/null || true"},
	}

	for _, diag := range diagnostics {
		if output, err := client.ExecuteCommand(diag.cmd); err != nil {
			logger.Warn(fmt.Sprintf("%s failed: %s", diag.name, err.Error()))
		} else {
			logger.Info(fmt.Sprintf("%s result: %s", diag.name, strings.TrimSpace(output)))
		}
	}

	// Try to fix repository configuration first
	logger.Info("Setting up proper Debian repositories...")
	if err := setupDebianRepositories(client); err != nil {
		logger.Warn("Failed to setup repositories: " + err.Error())
	}

	// Remove any locks
	logger.Info("Removing package manager locks...")
	lockCommands := []string{
		"rm -f /var/lib/apt/lists/lock",
		"rm -f /var/cache/apt/archives/lock",
		"rm -f /var/lib/dpkg/lock*",
		"dpkg --configure -a",
	}
	for _, cmd := range lockCommands {
		if _, err := client.ExecuteCommand(cmd); err != nil {
			logger.Warn("Lock removal command failed: " + err.Error())
		}
	}

	// Try update with retries
	logger.Info("Attempting package update with retries...")
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		logger.Info(fmt.Sprintf("Update attempt %d/%d", attempt, maxRetries))

		updateCommands := []string{
			"apt-get update -y",
			"apt-get update --fix-missing -y",
			"apt-get update --allow-releaseinfo-change -y",
		}

		updateSuccess := false
		for _, updateCmd := range updateCommands {
			if output, err := client.ExecuteCommand(updateCmd); err != nil {
				logger.Warn(fmt.Sprintf("Command '%s' failed: %s", updateCmd, err.Error()))
				logger.Warn(fmt.Sprintf("Output: %s", output))
			} else {
				logger.Info(fmt.Sprintf("Update successful with: %s", updateCmd))
				updateSuccess = true
				break
			}
		}

		if updateSuccess {
			return nil
		}

		if attempt == maxRetries {
			// Try one final fallback method
			logger.Info("Attempting final fallback update method...")
			return tryFallbackUpdate(client)
		}

		// Wait before retry
		logger.Info("Waiting 10 seconds before retry...")
		if _, err := client.ExecuteCommand("sleep 10"); err != nil {
			logger.Warn("Sleep command failed: " + err.Error())
		}
	}

	return fmt.Errorf("all update attempts failed")
}

// tryFallbackUpdate attempts alternative update methods when standard methods fail
func tryFallbackUpdate(client *SSHClient) error {
	logger.Info("Trying fallback update methods...")

	// Try to use different mirrors
	fallbackSources := `# Fallback Debian 12 repositories with different mirrors
deb http://httpredir.debian.org/debian bookworm main contrib non-free-firmware
deb http://ftp.us.debian.org/debian bookworm main contrib non-free-firmware
deb http://security.debian.org/debian-security bookworm-security main contrib non-free-firmware`

	// Write fallback sources
	writeSourcesCmd := fmt.Sprintf("cat > /etc/apt/sources.list << 'EOF'\n%s\nEOF", fallbackSources)
	if _, err := client.ExecuteCommand(writeSourcesCmd); err != nil {
		logger.Warn("Failed to write fallback sources: " + err.Error())
	}

	// Clear cache and try update
	fallbackCommands := []string{
		"apt-get clean",
		"rm -rf /var/lib/apt/lists/*",
		"apt-get update -o Acquire::Check-Valid-Until=false",
	}

	for _, cmd := range fallbackCommands {
		if output, err := client.ExecuteCommand(cmd); err != nil {
			logger.Warn(fmt.Sprintf("Fallback command failed: %s - %s", cmd, err.Error()))
			if output != "" {
				logger.Warn(fmt.Sprintf("Output: %s", output))
			}
		}
	}

	// Final test
	if output, err := client.ExecuteCommand("apt-get update"); err != nil {
		logger.Warn("Final fallback test output: " + output)

		// Check if it's a temporary network issue
		if strings.Contains(output, "Temporary failure resolving") ||
			strings.Contains(output, "Could not resolve") ||
			strings.Contains(output, "Network is unreachable") {
			return fmt.Errorf("network connectivity issue - please try again in a few minutes: %w", err)
		}

		// Check if it's a repository issue
		if strings.Contains(output, "404") ||
			strings.Contains(output, "Release file") ||
			strings.Contains(output, "not found") {
			return fmt.Errorf("repository configuration issue - manual intervention may be required: %w", err)
		}

		return fmt.Errorf("all fallback update methods failed: %w", err)
	}

	logger.Info("Fallback update method succeeded")
	return nil
}

// setupDebianRepositories ensures proper Debian repository configuration
func setupDebianRepositories(client *SSHClient) error {
	logger.Info("Configuring Debian repositories...")

	// Create a proper sources.list for Debian 12 (bookworm)
	debianSources := `# Debian 12 (bookworm) main repositories
deb http://deb.debian.org/debian bookworm main contrib non-free-firmware
deb-src http://deb.debian.org/debian bookworm main contrib non-free-firmware

# Security updates
deb http://security.debian.org/debian-security bookworm-security main contrib non-free-firmware
deb-src http://security.debian.org/debian-security bookworm-security main contrib non-free-firmware

# Updates
deb http://deb.debian.org/debian bookworm-updates main contrib non-free-firmware
deb-src http://deb.debian.org/debian bookworm-updates main contrib non-free-firmware`

	// Backup original sources.list
	if _, err := client.ExecuteCommand("cp /etc/apt/sources.list /etc/apt/sources.list.backup"); err != nil {
		logger.Warn("Failed to backup sources.list: " + err.Error())
	}

	// Write new sources.list
	writeSourcesCmd := fmt.Sprintf("cat > /etc/apt/sources.list << 'EOF'\n%s\nEOF", debianSources)
	if _, err := client.ExecuteCommand(writeSourcesCmd); err != nil {
		return fmt.Errorf("failed to write sources.list: %w", err)
	}

	logger.Info("Debian repositories configured successfully")
	return nil
}

// fixPackageIssues attempts to fix common package installation issues
func fixPackageIssues(client *SSHClient) error {
	fixCommands := []struct {
		name string
		cmd  string
	}{
		{"Fix broken packages", "apt-get -f install -y"},
		{"Configure pending packages", "dpkg --configure -a"},
		{"Clean package cache", "apt-get clean"},
		{"Remove package locks", "rm -f /var/lib/apt/lists/lock /var/cache/apt/archives/lock /var/lib/dpkg/lock*"},
		{"Update package lists", "apt-get update --fix-missing"},
	}

	for _, fix := range fixCommands {
		logger.Info("Attempting: " + fix.name)
		if output, err := client.ExecuteCommand(fix.cmd); err != nil {
			logger.Warn(fmt.Sprintf("%s failed: %s", fix.name, err.Error()))
			if output != "" {
				logger.Warn("Output: " + strings.TrimSpace(output))
			}
		} else {
			logger.Info(fmt.Sprintf("%s completed successfully", fix.name))
		}
	}

	return nil
}

// runPackageInstallDiagnostics runs diagnostic commands to help troubleshoot package installation issues
func runPackageInstallDiagnostics(client *SSHClient) error {
	diagnosticCommands := []struct {
		name string
		cmd  string
	}{
		{"Check disk space", "df -h"},
		{"Check memory", "free -h"},
		{"Check apt sources", "cat /etc/apt/sources.list"},
		{"Check for broken packages", "apt-get check"},
		{"Check for held packages", "apt-mark showhold"},
	}

	for _, diag := range diagnosticCommands {
		if output, err := client.ExecuteCommand(diag.cmd); err != nil {
			logger.Warn(fmt.Sprintf("%s failed: %s", diag.name, err.Error()))
		} else {
			logger.Info(fmt.Sprintf("%s: %s", diag.name, strings.TrimSpace(output)))
		}
	}

	return nil
}

// generateWireGuardKeys generates server and client key pairs
func generateWireGuardKeys(client *SSHClient) (*WireGuardKeys, error) {
	// Generate server keys
	serverPrivate, err := client.ExecuteCommand("wg genkey")
	if err != nil {
		return nil, fmt.Errorf("failed to generate server private key: %w", err)
	}
	serverPrivate = strings.TrimSpace(serverPrivate)

	serverPublic, err := client.ExecuteCommand(fmt.Sprintf("echo '%s' | wg pubkey", serverPrivate))
	if err != nil {
		return nil, fmt.Errorf("failed to generate server public key: %w", err)
	}
	serverPublic = strings.TrimSpace(serverPublic)

	// Generate client keys
	clientPrivate, err := client.ExecuteCommand("wg genkey")
	if err != nil {
		return nil, fmt.Errorf("failed to generate client private key: %w", err)
	}
	clientPrivate = strings.TrimSpace(clientPrivate)

	clientPublic, err := client.ExecuteCommand(fmt.Sprintf("echo '%s' | wg pubkey", clientPrivate))
	if err != nil {
		return nil, fmt.Errorf("failed to generate client public key: %w", err)
	}
	clientPublic = strings.TrimSpace(clientPublic)

	return &WireGuardKeys{
		ServerPrivate: serverPrivate,
		ServerPublic:  serverPublic,
		ClientPrivate: clientPrivate,
		ClientPublic:  clientPublic,
	}, nil
}

// downloadClientConfig downloads the WireGuard client configuration from the VPS
func downloadClientConfig(client *SSHClient, config PostProvisionConfig) error {
	logger.Info("Downloading WireGuard client configuration...")

	// Create local directory for client configs
	clientDir := "wireguard-clients"
	if err := os.MkdirAll(clientDir, 0755); err != nil {
		return fmt.Errorf("failed to create client directory: %w", err)
	}

	// Download client config using the utility function
	localConfigPath := filepath.Join(clientDir, fmt.Sprintf("client-%s.conf", config.IP))
	if err := DownloadFile(client, "/root/wireguard-client/client.conf", localConfigPath); err != nil {
		return fmt.Errorf("failed to download client config: %w", err)
	}

	logger.Info("WireGuard client configuration downloaded to: " + logger.Highlight(localConfigPath))

	// Also try to download QR code if available
	qrPath := filepath.Join(clientDir, fmt.Sprintf("client-%s-qr.txt", config.IP))
	if err := DownloadFile(client, "/root/wireguard-client/client-qr.txt", qrPath); err == nil {
		logger.Info("QR code saved to: " + logger.Highlight(qrPath))
	}

	return nil
}
