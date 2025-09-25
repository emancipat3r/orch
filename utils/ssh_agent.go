package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/emancipat3r/vps3/logger"
)

// AddKeyToSSHAgent adds an SSH key to ssh-agent with its passphrase
func AddKeyToSSHAgent(privKeyPath string) error {
	// Check if ssh-agent is running
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		logger.Debug("ssh-agent is not running, skipping key addition")
		return nil // Not an error, just skip
	}

	// Check if key is already in agent
	if CheckKeyInSSHAgent(privKeyPath) {
		logger.Debug("Key already in ssh-agent: " + privKeyPath)
		return nil
	}

	// Check if key has passphrase
	keyName := filepath.Base(privKeyPath)
	passFile := filepath.Join(filepath.Dir(filepath.Dir(privKeyPath)), "secrets", keyName+".pass")

	// Try to read passphrase
	passBytes, err := os.ReadFile(passFile)
	if err != nil {
		// No passphrase file, try adding without passphrase
		logger.Debug("No passphrase file found, attempting to add key without passphrase")
		cmd := exec.Command("ssh-add", privKeyPath)
		cmd.Stdin = nil
		if err := cmd.Run(); err != nil {
			logger.Debug(fmt.Sprintf("Failed to add key without passphrase: %v", err))
			return nil // Not critical, continue without agent
		}
		logger.Info("Added SSH key to ssh-agent: " + privKeyPath)
		return nil
	}

	passphrase := strings.TrimSpace(string(passBytes))

	// Check if sshpass is available for a cleaner approach
	if _, err := exec.LookPath("sshpass"); err == nil {
		// Use sshpass to provide passphrase
		cmd := exec.Command("sshpass", "-p", passphrase, "ssh-add", privKeyPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			logger.Debug(fmt.Sprintf("Failed to add key with sshpass: %v - %s", err, output))
			return nil // Not critical
		}
		logger.Info("Added SSH key to ssh-agent: " + privKeyPath)
		return nil
	}

	// Use expect or SSH_ASKPASS mechanism
	// Create a temporary askpass script
	askpassScript := fmt.Sprintf("/tmp/ssh_askpass_%d_%d.sh", os.Getpid(), time.Now().Unix())
	askpassContent := fmt.Sprintf("#!/bin/sh\necho '%s'\n", passphrase)
	if err := os.WriteFile(askpassScript, []byte(askpassContent), 0700); err != nil {
		logger.Debug(fmt.Sprintf("Failed to create askpass script: %v", err))
		return nil
	}
	defer os.Remove(askpassScript)

	// Try SSH_ASKPASS method
	cmd := exec.Command("ssh-add", privKeyPath)
	cmd.Env = append(os.Environ(),
		"SSH_ASKPASS="+askpassScript,
		"SSH_ASKPASS_REQUIRE=force",
		"DISPLAY=:0", // Needed for SSH_ASKPASS to work
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		// Last resort: try using expect if available
		if _, err := exec.LookPath("expect"); err == nil {
			expectScript := fmt.Sprintf("/tmp/ssh_add_expect_%d.exp", os.Getpid())
			expectContent := fmt.Sprintf(`#!/usr/bin/expect -f
set timeout 5
spawn ssh-add %s
expect "Enter passphrase"
send "%s\r"
expect eof
`, privKeyPath, passphrase)
			if err := os.WriteFile(expectScript, []byte(expectContent), 0700); err == nil {
				defer os.Remove(expectScript)
				cmd := exec.Command("expect", expectScript)
				if err := cmd.Run(); err == nil {
					logger.Info("Added SSH key to ssh-agent: " + privKeyPath)
					return nil
				}
			}
		}

		logger.Debug(fmt.Sprintf("Could not add key to ssh-agent: %v - %s", err, output))
		logger.Debug("You can manually add it with: ssh-add " + privKeyPath)
		return nil // Not critical, continue without agent
	}

	logger.Info("Added SSH key to ssh-agent: " + privKeyPath)
	return nil
}

// RemoveKeyFromSSHAgent removes an SSH key from ssh-agent
func RemoveKeyFromSSHAgent(privKeyPath string) error {
	// Check if ssh-agent is running
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		logger.Debug("ssh-agent is not running, skipping key removal")
		return nil
	}

	// Check if key is in agent
	if !CheckKeyInSSHAgent(privKeyPath) {
		logger.Debug("Key not in ssh-agent, nothing to remove: " + privKeyPath)
		return nil
	}

	// Remove the key
	cmd := exec.Command("ssh-add", "-d", privKeyPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		logger.Debug(fmt.Sprintf("Failed to remove key from ssh-agent: %v - %s", err, output))
		return nil // Not critical
	}

	logger.Info("Removed SSH key from ssh-agent: " + privKeyPath)
	return nil
}

// EnsureSSHAgent ensures ssh-agent is running and returns whether it's available
func EnsureSSHAgent() bool {
	// Check if ssh-agent is already running
	if os.Getenv("SSH_AUTH_SOCK") != "" {
		logger.Debug("ssh-agent is already running")
		return true
	}

	// Try to start ssh-agent
	cmd := exec.Command("ssh-agent", "-s")
	output, err := cmd.Output()
	if err != nil {
		logger.Debug("Could not start ssh-agent: " + err.Error())
		return false
	}

	// Parse the output to set environment variables
	// Output looks like:
	// SSH_AUTH_SOCK=/tmp/ssh-XXX/agent.123; export SSH_AUTH_SOCK;
	// SSH_AGENT_PID=456; export SSH_AGENT_PID;
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "SSH_AUTH_SOCK=") {
			parts := strings.Split(line, ";")
			if len(parts) > 0 {
				envPart := strings.TrimPrefix(parts[0], "SSH_AUTH_SOCK=")
				os.Setenv("SSH_AUTH_SOCK", envPart)
			}
		} else if strings.HasPrefix(line, "SSH_AGENT_PID=") {
			parts := strings.Split(line, ";")
			if len(parts) > 0 {
				envPart := strings.TrimPrefix(parts[0], "SSH_AGENT_PID=")
				os.Setenv("SSH_AGENT_PID", envPart)
			}
		}
	}

	if os.Getenv("SSH_AUTH_SOCK") != "" {
		logger.Info("Started ssh-agent successfully")
		return true
	}

	logger.Debug("Could not configure ssh-agent environment")
	return false
}
