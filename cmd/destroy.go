package cmd

import (
	"strconv"

	"github.com/emancipat3r/orch/logger"
	"github.com/emancipat3r/orch/providers"
	"github.com/emancipat3r/orch/ui"
	"github.com/emancipat3r/orch/utils"
	"github.com/spf13/cobra"
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

		// Registry lookups are best-effort: an instance that exists remotely
		// but not locally is still destroyed, we just have nothing local to
		// clean up for it.
		db, err := utils.LoadInstances(instanceFile)
		if err != nil {
			logger.Warn("Could not read instance registry; local artifacts won't be cleaned up: " + err.Error())
			db = utils.InstanceDB{}
		}

		succeeded := 0
		for i, s := range selected {
			inst := byLabel[s]
			rec, tracked := db[inst.ID]
			if !tracked {
				logger.Warn("Instance " + inst.ID + " is not in the local registry; only the remote VPS will be removed")
			}

			logger.Info("(" + strconv.Itoa(i+1) + "/" + strconv.Itoa(len(selected)) +
				") Destroying " + providerName + " instance: " + logger.Highlight(inst.ID))

			if err := prov.Destroy(ctx, inst.ID, rec.SSHKeyID); err != nil {
				logger.Error("Failed to destroy " + providerName + " instance " + inst.ID + ": " + err.Error())
				logger.Info("Local artifacts for " + inst.ID + " were left in place; re-run destroy once the provider call succeeds")
				continue
			}
			succeeded++

			if !tracked {
				continue
			}
			if errs := utils.CleanupLocalArtifacts(paths, rec); len(errs) > 0 {
				logger.Warn(strconv.Itoa(len(errs)) + " local artifact(s) could not be removed; run `orch prune` to retry")
			}
			if err := utils.RemoveInstanceRecords(instanceFile, []string{inst.ID}); err != nil {
				logger.Error("Failed to remove " + inst.ID + " from registry: " + err.Error())
			}
		}

		logger.Info("Destroyed " + logger.Highlight(strconv.Itoa(succeeded)) + "/" + strconv.Itoa(len(selected)) + " " + providerName + " instance(s)")
	},
}

func init() {
	rootCmd.AddCommand(destroyCmd)
}
