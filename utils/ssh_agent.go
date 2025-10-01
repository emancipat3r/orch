package utils

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/emancipat3r/vps3/logger"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// AddKeyToSSHAgent adds an SSH key to ssh-agent with its passphrase
func AddKeyToSSHAgent(privKeyPath string) error {
	// 1) Ensure agent is available
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		logger.Debug("ssh-agent is not running (SSH_AUTH_SOCK empty); skipping key addition")
		return nil // match your original semantics: non-critical
	}

	// 2) Read private key
	keyPEM, err := os.ReadFile(privKeyPath)
	if err != nil {
		return logger.Errorf("read private key: %w", err)
	}

	// 3) Derive passphrase path and read passphrase if present
	passphrase, err := readPassphraseForKey(privKeyPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		// Only treat non-ENOENT as error; if the file doesn't exist, we try no-pass.
		return logger.Errorf("read passphrase: %w", err)
	}

	// 4) Parse the private key
	var parsed any
	if passphrase != "" {
		parsed, err = ssh.ParseRawPrivateKeyWithPassphrase(keyPEM, []byte(passphrase))
	} else {
		parsed, err = ssh.ParseRawPrivateKey(keyPEM)
	}
	if err != nil {
		return logger.Errorf("parse private key: %w", err)
	}

	// 5) Build signer so we can compare public keys and avoid duplicates
	signer, err := ssh.NewSignerFromKey(parsed)
	if err != nil {
		return logger.Errorf("make signer: %w", err)
	}
	wantBlob := signer.PublicKey().Marshal()

	// 6) Connect to the agent
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return logger.Errorf("dial ssh-agent: %w", err)
	}
	defer conn.Close()
	ag := agent.NewClient(conn)

	// 7) Skip if already loaded
	if alreadyLoaded(ag, wantBlob) {
		logger.Debug("Key already in ssh-agent: " + privKeyPath)
		return nil
	}

	// 8) Add to agent
	err = ag.Add(agent.AddedKey{
		PrivateKey:       parsed,
		Comment:          privKeyPath,
		LifetimeSecs:     0,
		ConfirmBeforeUse: false,
	})
	if err != nil {
		return fmt.Errorf("agent add: %w", err)
	}

	logger.Info("Added SSH key to ssh-agent: " + logger.Highlight(privKeyPath))
	return nil
}

func readPassphraseForKey(privKeyPath string) (string, error) {
	keyName := filepath.Base(privKeyPath)              // e.g., do-13072376
	cfgRoot := filepath.Dir(filepath.Dir(privKeyPath)) // .../.config/vps
	passFile := filepath.Join(cfgRoot, "secrets", keyName+".pass")

	b, err := os.ReadFile(passFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func alreadyLoaded(ag agent.ExtendedAgent, wantBlob []byte) bool {
	keys, err := ag.List()
	if err != nil {
		// If we can't list, conservatively assume not loaded so we try to add.
		return false
	}
	for _, k := range keys {
		if bytes.Equal(k.Blob, wantBlob) {
			return true
		}
	}
	return false
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

	logger.Info("Removed SSH key from ssh-agent: " + logger.Highlight(privKeyPath))
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
