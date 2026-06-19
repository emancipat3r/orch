package cmd

import (
	"context"
	"os"
	"strconv"

	"github.com/emancipat3r/orch/logger"
	"github.com/emancipat3r/orch/providers"
	"github.com/emancipat3r/orch/ui"
	"github.com/emancipat3r/orch/utils"
	"github.com/spf13/cobra"
)

var pruneRemote bool

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove local artifacts not referenced by the instance registry",
	Long: `Reconcile local state against the instance registry (instances.toml).

By default, prune removes on-disk SSH keys, passphrases, and WireGuard configs
that no registry entry references — the leftovers from failed creates or
instances destroyed out of band. It never touches anything still referenced by a
live registry entry.

With --remote, prune first queries each provider and removes registry entries
whose VPS no longer exists remotely; their local keys then become orphans and
are cleaned up in the same run.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		if pruneRemote {
			reconcileRemote(ctx)
		}

		orphans, err := utils.FindOrphans(instanceFile, pathSSH, pathSecrets, pathWg)
		if err != nil {
			logger.Error("Failed to scan for orphans: " + err.Error())
			return
		}

		if orphans.Total() == 0 {
			logger.Info("Nothing to prune — local artifacts match the instance registry.")
			return
		}

		logger.Warn("Found " + strconv.Itoa(orphans.Total()) + " orphaned artifact(s) not referenced by any instance:")
		for _, k := range orphans.SSHKeys {
			logger.Info("  ssh key   " + logger.Highlight(k))
		}
		for _, s := range orphans.Secrets {
			logger.Info("  secret    " + logger.Highlight(s))
		}
		for _, c := range orphans.WgConfigs {
			logger.Info("  wg config " + logger.Highlight(c))
		}

		if !ui.Confirm("Delete these " + strconv.Itoa(orphans.Total()) + " orphaned file(s)?") {
			logger.Info("Prune cancelled by user.")
			return
		}

		removed := 0
		for _, k := range orphans.SSHKeys {
			// Best-effort removal from ssh-agent before deleting the key on disk.
			if err := utils.RemoveKeyFromSSHAgent(k); err != nil {
				logger.Debug("Could not remove key from ssh-agent: " + err.Error())
			}
			if removeFile(k) {
				removed++
			}
			removeFile(k + ".pub") // counted with its private key, not separately
		}
		for _, s := range orphans.Secrets {
			if removeFile(s) {
				removed++
			}
		}
		for _, c := range orphans.WgConfigs {
			if removeFile(c) {
				removed++
			}
		}

		logger.Info("Pruned " + logger.Highlight(strconv.Itoa(removed)) + " artifact(s).")
	},
}

// reconcileRemote drops registry entries whose VPS no longer exists on the
// provider. Providers whose credentials are missing or whose list call fails are
// skipped (never deleted) so we only ever remove entries we could verify are gone.
func reconcileRemote(ctx context.Context) {
	if _, err := os.Stat(instanceFile); err != nil {
		return // no registry yet, nothing to reconcile
	}
	db, err := utils.LoadInstances(instanceFile)
	if err != nil {
		logger.Warn("Could not read registry for remote reconcile: " + err.Error())
		return
	}
	if len(db) == 0 {
		return
	}

	byProvider := map[string][]utils.InstanceRecord{}
	for _, rec := range db {
		if rec.Provider == "" || rec.Id == "" {
			continue
		}
		byProvider[rec.Provider] = append(byProvider[rec.Provider], rec)
	}

	var stale []string
	for provName, recs := range byProvider {
		prov, err := providers.GetProvider(provName, configFile)
		if err != nil {
			logger.Warn("Skipping " + provName + " (no usable credentials): " + err.Error())
			continue
		}
		live, err := prov.List(ctx)
		if err != nil {
			logger.Warn("Skipping " + provName + " (list failed): " + err.Error())
			continue
		}
		liveIDs := map[string]bool{}
		for _, inst := range live {
			liveIDs[inst.ID] = true
		}
		for _, rec := range recs {
			if !liveIDs[rec.Id] {
				logger.Info("Registry entry " + logger.Highlight(rec.Id) + " (" + provName + ") no longer exists remotely")
				stale = append(stale, rec.Id)
			}
		}
	}

	if len(stale) == 0 {
		logger.Info("Registry is in sync with all reachable providers.")
		return
	}

	if !ui.Confirm("Remove " + strconv.Itoa(len(stale)) + " stale registry entr(y/ies)?") {
		logger.Info("Keeping registry entries.")
		return
	}

	if err := utils.RemoveInstanceRecords(instanceFile, stale); err != nil {
		logger.Error("Failed to update registry: " + err.Error())
		return
	}
	logger.Info("Removed " + strconv.Itoa(len(stale)) + " stale registry entr(y/ies); their local keys will be pruned below.")
}

// removeFile deletes path, logging a warning only on a real (non-absent) error.
// Returns true if the file was actually removed.
func removeFile(path string) bool {
	if err := os.Remove(path); err != nil {
		if !os.IsNotExist(err) {
			logger.Warn("Failed to delete " + path + ": " + err.Error())
		}
		return false
	}
	return true
}

func init() {
	pruneCmd.Flags().BoolVar(&pruneRemote, "remote", false, "Also query providers and drop registry entries whose VPS no longer exists")
	rootCmd.AddCommand(pruneCmd)
}
