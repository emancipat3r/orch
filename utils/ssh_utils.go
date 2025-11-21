package utils

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// IsPortReachable checks if a TCP port is reachable (like netcat)
func IsPortReachable(host string, port string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
