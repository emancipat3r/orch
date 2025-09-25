package utils

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emancipat3r/vps3/logger"
	"golang.org/x/crypto/ssh"
)

// SSHClient wraps an SSH connection with helper methods
type SSHClient struct {
	*ssh.Client
	Config *ssh.ClientConfig
	Host   string
	Port   string
}

// NewSSHClient creates a new SSH client with the given configuration
func NewSSHClient(host, port, user, privKeyPath string) (*SSHClient, error) {
	// Read the private key
	keyBytes, err := os.ReadFile(privKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	// Check if key has passphrase
	keyName := filepath.Base(privKeyPath)
	passFile := filepath.Join(filepath.Dir(filepath.Dir(privKeyPath)), "secrets", keyName+".pass")
	var signer ssh.Signer

	if passBytes, err := os.ReadFile(passFile); err == nil {
		passphrase := strings.TrimSpace(string(passBytes))
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(passphrase))
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key with passphrase: %w", err)
		}
	} else {
		signer, err = ssh.ParsePrivateKey(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
	}

	// SSH configuration
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	return &SSHClient{
		Config: config,
		Host:   host,
		Port:   port,
	}, nil
}

// Connect establishes the SSH connection
func (c *SSHClient) Connect() error {
	client, err := ssh.Dial("tcp", net.JoinHostPort(c.Host, c.Port), c.Config)
	if err != nil {
		return fmt.Errorf("failed to connect to %s:%s: %w", c.Host, c.Port, err)
	}
	c.Client = client
	return nil
}

// Close closes the SSH connection safely
func (c *SSHClient) Close() error {
	if c.Client != nil {
		return c.Client.Close()
	}
	return nil
}

// ExecuteCommand runs a command on the remote server
func (c *SSHClient) ExecuteCommand(command string) (string, error) {
	if c.Client == nil {
		return "", fmt.Errorf("SSH client not connected")
	}

	session, err := c.Client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("command '%s' failed: %w", command, err)
	}

	return string(output), nil
}

// ExecuteCommandWithTimeout runs a command with a timeout
func (c *SSHClient) ExecuteCommandWithTimeout(command string, timeout time.Duration) (string, error) {
	if c.Client == nil {
		return "", fmt.Errorf("SSH client not connected")
	}

	session, err := c.Client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Create a channel to receive the result
	resultChan := make(chan struct {
		output string
		err    error
	}, 1)

	// Run command in a goroutine
	go func() {
		output, err := session.CombinedOutput(command)
		resultChan <- struct {
			output string
			err    error
		}{string(output), err}
	}()

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		if result.err != nil {
			return result.output, fmt.Errorf("command '%s' failed: %w", command, result.err)
		}
		return result.output, nil
	case <-time.After(timeout):
		return "", fmt.Errorf("command '%s' timed out after %v", command, timeout)
	}
}

// TestConnectivity tests if SSH connectivity is working
func TestSSHConnectivity(host, port, user, privKeyPath string) error {
	logger.Info("Testing SSH connectivity to " + logger.Highlight(host+":"+port))

	client, err := NewSSHClient(host, port, user, privKeyPath)
	if err != nil {
		return fmt.Errorf("failed to create SSH client: %w", err)
	}

	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	// Test with a simple command
	output, err := client.ExecuteCommand("echo 'SSH connectivity test successful'")
	if err != nil {
		return fmt.Errorf("failed to execute test command: %w", err)
	}

	if !strings.Contains(output, "SSH connectivity test successful") {
		return fmt.Errorf("unexpected test command output: %s", output)
	}

	logger.Info("SSH connectivity test passed")
	return nil
}

// GetSystemInfo retrieves basic system information
func GetSystemInfo(client *SSHClient) (map[string]string, error) {
	info := make(map[string]string)

	// Get OS information
	if output, err := client.ExecuteCommand("cat /etc/os-release | grep '^ID=' | cut -d= -f2 | tr -d '\"'"); err == nil {
		info["os_id"] = strings.TrimSpace(output)
	}

	if output, err := client.ExecuteCommand("cat /etc/os-release | grep '^VERSION_ID=' | cut -d= -f2 | tr -d '\"'"); err == nil {
		info["os_version"] = strings.TrimSpace(output)
	}

	// Get hostname
	if output, err := client.ExecuteCommand("hostname"); err == nil {
		info["hostname"] = strings.TrimSpace(output)
	}

	// Get uptime
	if output, err := client.ExecuteCommand("uptime"); err == nil {
		info["uptime"] = strings.TrimSpace(output)
	}

	// Get kernel version
	if output, err := client.ExecuteCommand("uname -r"); err == nil {
		info["kernel"] = strings.TrimSpace(output)
	}

	// Check if WireGuard is installed
	if _, err := client.ExecuteCommand("which wg"); err == nil {
		info["wireguard"] = "installed"
	} else {
		info["wireguard"] = "not_installed"
	}

	// Check if WireGuard service is running
	if output, err := client.ExecuteCommand("systemctl is-active wg-quick@wg0"); err == nil {
		info["wireguard_status"] = strings.TrimSpace(output)
	} else {
		info["wireguard_status"] = "inactive"
	}

	return info, nil
}

// CreateRemoteFile creates a file on the remote server with specified content and permissions
func CreateRemoteFile(client *SSHClient, remotePath, content, permissions string) error {
	// Escape single quotes in content for safe shell usage
	escapedContent := strings.ReplaceAll(content, "'", "'\"'\"'")

	// Write content to file using printf to handle special characters properly
	cmd := fmt.Sprintf("printf '%%s' '%s' > %s", escapedContent, remotePath)
	if _, err := client.ExecuteCommand(cmd); err != nil {
		return fmt.Errorf("failed to write file %s: %w", remotePath, err)
	}

	// Set file permissions if specified
	if permissions != "" {
		if _, err := client.ExecuteCommand(fmt.Sprintf("chmod %s %s", permissions, remotePath)); err != nil {
			return fmt.Errorf("failed to set permissions on %s: %w", remotePath, err)
		}
	}

	return nil
}

// CreateRemoteDirectory creates a directory on the remote server with specified permissions
func CreateRemoteDirectory(client *SSHClient, remotePath, permissions string) error {
	cmd := fmt.Sprintf("mkdir -p %s", remotePath)
	if _, err := client.ExecuteCommand(cmd); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", remotePath, err)
	}

	// Set directory permissions if specified
	if permissions != "" {
		if _, err := client.ExecuteCommand(fmt.Sprintf("chmod %s %s", permissions, remotePath)); err != nil {
			return fmt.Errorf("failed to set permissions on directory %s: %w", remotePath, err)
		}
	}

	return nil
}

// IsPortReachable checks if a TCP port is reachable (like netcat)
func IsPortReachable(host string, port string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// CheckPortOpen checks if a specific port is open and listening
func CheckPortOpen(client *SSHClient, port int) (bool, error) {
	cmd := fmt.Sprintf("netstat -tuln | grep ':%d '", port)
	output, err := client.ExecuteCommand(cmd)
	if err != nil {
		// Command might fail if netstat isn't available, try ss instead
		cmd = fmt.Sprintf("ss -tuln | grep ':%d '", port)
		output, err = client.ExecuteCommand(cmd)
		if err != nil {
			return false, fmt.Errorf("failed to check port %d: %w", port, err)
		}
	}

	return strings.TrimSpace(output) != "", nil
}

// WaitForPort waits for a specific port to become available
func WaitForPort(client *SSHClient, port int, timeout time.Duration) error {
	logger.Info(fmt.Sprintf("Waiting for port %d to become available...", port))

	start := time.Now()
	for time.Since(start) < timeout {
		if isOpen, err := CheckPortOpen(client, port); err == nil && isOpen {
			logger.Info(fmt.Sprintf("Port %d is now available", port))
			return nil
		}
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("timeout waiting for port %d to become available after %v", port, timeout)
}

// DownloadFile downloads a file from the remote server
func DownloadFile(client *SSHClient, remotePath, localPath string) error {
	// Read the remote file content
	content, err := client.ExecuteCommand(fmt.Sprintf("cat %s", remotePath))
	if err != nil {
		return fmt.Errorf("failed to read remote file %s: %w", remotePath, err)
	}

	// Write content to local file
	if err := os.WriteFile(localPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write local file %s: %w", localPath, err)
	}

	return nil
}
