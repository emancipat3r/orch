package utils

import (
	"os"

	"github.com/emancipat3r/orch/logger"
)

// CleanupLocalArtifacts removes everything orch created on this machine for an
// instance: the ssh-agent identity, private/public key, passphrase, WireGuard
// client config, and the wg-<name> interface + network namespace. Every step is
// best-effort and idempotent so it can be run against a half-created instance
// or repeated safely; anything that genuinely failed to delete is returned so
// the caller can surface it. Registry removal is deliberately separate.
func CleanupLocalArtifacts(p Paths, rec InstanceRecord) []error {
	var errs []error
	remove := func(what, path string) {
		if path == "" {
			return
		}
		err := os.Remove(path)
		switch {
		case err == nil:
			logger.Info("Deleted " + what + ": " + logger.Highlight(path))
		case os.IsNotExist(err):
			// already gone — nothing to report
		default:
			logger.Warn("Failed to delete " + what + " " + path + ": " + err.Error())
			errs = append(errs, err)
		}
	}

	if rec.PrivKeyPath != "" {
		if err := RemoveKeyFromSSHAgent(rec.PrivKeyPath); err != nil {
			logger.Debug("Could not remove key from ssh-agent: " + err.Error())
		}
		remove("private key", rec.PrivKeyPath)
		remove("public key", rec.PrivKeyPath+".pub")
		remove("passphrase file", p.PassFile(rec.PrivKeyPath))
	}

	// Both the named and the legacy client-<ip> config may exist for an
	// instance that was created before names were introduced and then re-run
	// through setup; remove whichever are present.
	remove("WireGuard config", p.WgConf(rec.VPSName, ""))
	remove("WireGuard config", p.WgConf("", rec.Ipv4))

	if rec.VPSName != "" {
		CleanupWireGuard("wg-"+rec.VPSName, rec.VPSName)
	}
	return errs
}
