package cmd

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/emancipat3r/orch/logger"
	"github.com/emancipat3r/orch/providers"
	"github.com/emancipat3r/orch/ui"
	"github.com/emancipat3r/orch/utils"
	"github.com/spf13/cobra"
	"github.com/vishvananda/netns"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy VPS instance(s)",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		providerName := ui.ChoiceProvider()
		if providerName == "" {
			logger.Info("Operation cancelled by user.")
			return
		}
		if ctx.Err() != nil {
			return
		}

		prov, err := providers.GetProvider(providerName, configFile)
		if err != nil {
			logger.Error(err.Error())
			return
		}

		balance, err := prov.Balance(ctx)
		if err != nil {
			logger.Error("Failed to get " + providerName + " account balance: " + err.Error())
			return
		}
		logger.Info(providerName + " account balance: " + logger.Highlight("$"+balance))

		instances, err := prov.List(ctx)
		if err != nil {
			logger.Error("Failed to list instances: " + err.Error())
			return
		}
		if len(instances) == 0 {
			logger.Info("No instances found. Exiting...")
			return
		}

		// Map each display label back to its instance so selection never relies
		// on positional string parsing of the rendered label.
		options := make([]string, 0, len(instances))
		byLabel := make(map[string]providers.Instance, len(instances))
		for _, inst := range instances {
			label := inst.Created + " - " + inst.ID + " - " + inst.Image + " - " + inst.IP + " - " + inst.Region
			options = append(options, label)
			byLabel[label] = inst
		}

		selected := ui.MultiSelect("Select the instance(s) to destroy:", options)
		if len(selected) == 0 {
			logger.Info("No instances selected. Exiting...")
			return
		}

		logger.Info("You selected " + strconv.Itoa(len(selected)) + " instance(s) for destruction")
		for _, s := range selected {
			logger.Info("  - " + logger.Highlight(s))
		}

		if !ui.Confirm("Are you sure you want to proceed with destroying " + strconv.Itoa(len(selected)) + " instance(s)?") {
			logger.Info("Destruction cancelled by user.")
			return
		}

		for i, s := range selected {
			inst := byLabel[s]

			// Resolve vps_name BEFORE destroying so we can tear down the local
			// WireGuard/netns afterwards.
			vpsName, err := utils.GetVPSNameForInstance(instanceFile, inst.ID)
			if err != nil {
				logger.Debug("No local vps_name mapping for " + providerName + " instance " + inst.ID + ": " + err.Error())
			}

			logger.Info("(" + strconv.Itoa(i+1) + "/" + strconv.Itoa(len(selected)) +
				") Destroying " + providerName + " instance: " + logger.Highlight(inst.ID))

			if err := prov.Destroy(ctx, inst.ID, instanceFile); err != nil {
				logger.Error("Failed to destroy " + providerName + " instance " + inst.ID + ": " + err.Error())
				continue
			}

			logger.Info("Successfully destroyed " + providerName + " instance: " + logger.Highlight(inst.ID))

			if vpsName != "" {
				tearDownLocalByName(vpsName)
			}
		}

		logger.Info("Completed destruction of " + logger.Highlight(strconv.Itoa(len(selected))) + " " + providerName + " instance(s)")
	},
}

func tearDownLocalByName(nsName string) {
	logger.Info("Tearing down local WireGuard/netns for " + logger.Highlight(nsName))

	if err := utils.TearDownNamespace(nsName, netns.None()); err != nil {
		logger.Error("Failed to tear down netns " + nsName + ": " + err.Error())
	}

	nsDir := filepath.Join("/etc/netns", nsName)
	_ = os.Remove(filepath.Join(nsDir, "resolv.conf"))
	_ = os.Remove(nsDir)

	_ = os.Remove(paths.WgConf(nsName, ""))

	logger.Info("Local WireGuard/netns cleaned up for " + logger.Highlight(nsName))
}

func init() {
	rootCmd.AddCommand(destroyCmd)
}
