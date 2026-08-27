package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// Paths is the single source of truth for orch's on-disk layout under
// ~/.config/orch. Every command and helper that touches local state should
// derive locations from here rather than re-joining home + ".config/orch/…" by
// hand, so the directory set stays consistent across create/setup/list/destroy/
// prune and can be created uniformly by Ensure.
type Paths struct {
	Root      string // ~/.config/orch
	Config    string // provider API keys
	SSH       string // per-instance private/public keys
	Secrets   string // per-instance key passphrases (.pass)
	Instances string // instance registry
	Ansible   string // generated inventory
	Wg        string // downloaded WireGuard client configs (.conf)

	InstanceFile string // Instances/instances.toml
	ConfigFile   string // Config/configuration.toml
}

// OrchPaths resolves the layout for the current user's home directory.
func OrchPaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("resolve home directory: %w", err)
	}
	return PathsUnder(filepath.Join(home, ".config", "orch")), nil
}

// PathsUnder builds the layout rooted at root. Exposed so tests can point orch
// at a temp directory.
func PathsUnder(root string) Paths {
	p := Paths{
		Root:      root,
		Config:    filepath.Join(root, "config"),
		SSH:       filepath.Join(root, ".ssh"),
		Secrets:   filepath.Join(root, "secrets"),
		Instances: filepath.Join(root, "instances"),
		Ansible:   filepath.Join(root, "ansible"),
		Wg:        filepath.Join(root, "wg"),
	}
	p.InstanceFile = filepath.Join(p.Instances, "instances.toml")
	p.ConfigFile = filepath.Join(p.Config, "configuration.toml")
	return p
}

// Dirs lists every directory orch requires, in creation order.
func (p Paths) Dirs() []string {
	return []string{p.Config, p.SSH, p.Secrets, p.Instances, p.Ansible, p.Wg}
}

// Ensure creates any missing required directory. It is safe to call from any
// code path (not just the root pre-run hook) so helpers invoked out of order —
// or from tests — never write into a directory that doesn't exist yet.
// onCreate, if non-nil, is called for each directory that had to be created.
func (p Paths) Ensure(onCreate func(dir string)) error {
	for _, dir := range p.Dirs() {
		if DirExists(dir) {
			continue
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
		if onCreate != nil {
			onCreate(dir)
		}
	}
	return nil
}

// WgConf is the local client config path for an instance. vpsName is preferred;
// the "client-<ip>" form is the legacy name used before instances had names.
func (p Paths) WgConf(vpsName, ip string) string {
	switch {
	case vpsName != "":
		return filepath.Join(p.Wg, vpsName+".conf")
	case ip != "" && ip != "pending":
		return filepath.Join(p.Wg, "client-"+ip+".conf")
	default:
		return ""
	}
}

// PassFile is the passphrase file for the private key at privKeyPath.
func (p Paths) PassFile(privKeyPath string) string {
	return filepath.Join(p.Secrets, filepath.Base(privKeyPath)+".pass")
}
