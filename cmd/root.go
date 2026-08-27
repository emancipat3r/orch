package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/emancipat3r/orch/embedded"
	"github.com/emancipat3r/orch/logger"
	"github.com/emancipat3r/orch/utils"
	"github.com/spf13/cobra"
)

var (
	pathConfig    string
	pathSSH       string
	pathSecrets   string
	pathInstances string
	pathWg        string
	instanceFile  string
	configFile    string
	pathAnsible   string
	vpsName       string
	paths         utils.Paths // canonical layout; the string globals above are derived from it
)

// rootCmd is the base command for the CLI
var rootCmd = &cobra.Command{
	Use:   "orch",
	Short: "VPS management CLI",
	Long:  "A CLI tool for provisioning and managing VPS instances.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// help/completion don't touch the filesystem — bail before any I/O.
		if cmd.Name() == "help" || cmd.Name() == "completion" {
			return nil
		}
		return ensureEnvironment()
	},
}

// ensureEnvironment populates the path globals, creates the orch config tree if
// missing, and seeds configuration.toml from the embedded template on first run.
// It runs for every command other than help/completion so first-time users can
// invoke any subcommand without manually bootstrapping ~/.config/orch.
func ensureEnvironment() error {
	p, err := utils.OrchPaths()
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	paths = p
	sep := string(filepath.Separator)
	pathConfig = p.Config + sep
	pathSSH = p.SSH + sep
	pathSecrets = p.Secrets + sep
	pathInstances = p.Instances + sep
	pathAnsible = p.Ansible + sep
	pathWg = p.Wg + sep
	instanceFile = p.InstanceFile
	configFile = p.ConfigFile

	if err := p.Ensure(func(dir string) {
		logger.Info("Created missing directory: " + logger.Highlight(dir))
	}); err != nil {
		logger.Error("Failed to create directory: " + err.Error())
		return err
	}

	if utils.FileExists(configFile) {
		return nil
	}

	logger.Warn("Provider configuration file missing.")
	templateData, err := embedded.Assets.ReadFile("templates/configuration.toml")
	if err != nil {
		logger.Error("Configuration template not found in embedded assets")
		return fmt.Errorf("missing config file: %s", configFile)
	}

	logger.Info("Copying configuration template to: " + logger.Highlight(configFile))
	if err := os.WriteFile(configFile, templateData, 0644); err != nil {
		return fmt.Errorf("failed to write config template: %w", err)
	}
	logger.Info("Configuration template copied successfully!")
	logger.Warn("Please edit " + logger.Highlight(configFile) + " and add your provider API keys")
	return fmt.Errorf("configuration file created - please add your API keys and try again")
}

// installSignalHandler wires SIGINT/SIGTERM to context cancellation so commands
// can return through their normal control flow instead of being killed mid-API
// call by os.Exit. The second signal forces an immediate exit with code 130 in
// case an in-flight operation isn't honoring the context.
func installSignalHandler(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Warn("Interrupt received — cancelling. Press Ctrl+C again to force exit.")
		cancel()
		<-sigCh
		fmt.Fprintln(os.Stderr, "Forcing exit.")
		os.Exit(130)
	}()
}

func Execute() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	installSignalHandler(cancel)
	rootCmd.SetContext(ctx)

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		// Don't print "context canceled" as a stack-trace-worthy error; that
		// just means the user pressed Ctrl+C.
		if !errors.Is(err, context.Canceled) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
