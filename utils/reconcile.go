package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// Orphans holds local artifacts that exist on disk but are not referenced by any
// entry in the instance registry — leftovers from failed creates or instances
// destroyed out of band.
type Orphans struct {
	SSHKeys   []string // private key paths (the matching .pub follows it)
	Secrets   []string // passphrase (.pass) files
	WgConfigs []string // WireGuard client (.conf) files
}

// Total returns the number of orphaned artifacts found.
func (o Orphans) Total() int {
	return len(o.SSHKeys) + len(o.Secrets) + len(o.WgConfigs)
}

// FindOrphans scans the per-instance SSH key, secret, and WireGuard directories
// and returns every artifact not referenced by the registry at instanceFile. A
// missing registry file is treated as an empty registry (everything is orphaned).
func FindOrphans(instanceFile, sshDir, secretsDir, wgDir string) (Orphans, error) {
	db := map[string]InstanceRecord{}
	if _, err := os.Stat(instanceFile); err == nil {
		if _, err := toml.DecodeFile(instanceFile, &db); err != nil {
			return Orphans{}, fmt.Errorf("decode %s: %w", instanceFile, err)
		}
	}

	keys := map[string]bool{}   // referenced private-key basenames
	passes := map[string]bool{} // referenced .pass basenames
	confs := map[string]bool{}  // referenced .conf basenames
	for _, rec := range db {
		if rec.PrivKeyPath != "" {
			base := filepath.Base(rec.PrivKeyPath)
			keys[base] = true
			passes[base+".pass"] = true
		}
		if rec.VPSName != "" {
			confs[rec.VPSName+".conf"] = true
		}
	}

	var o Orphans
	for _, name := range listFiles(sshDir) {
		// .pub files are deleted alongside their private key, not on their own.
		if strings.HasSuffix(name, ".pub") {
			continue
		}
		if !keys[name] {
			o.SSHKeys = append(o.SSHKeys, filepath.Join(sshDir, name))
		}
	}
	for _, name := range listFiles(secretsDir) {
		if !strings.HasSuffix(name, ".pass") {
			continue
		}
		if !passes[name] {
			o.Secrets = append(o.Secrets, filepath.Join(secretsDir, name))
		}
	}
	for _, name := range listFiles(wgDir) {
		if !strings.HasSuffix(name, ".conf") {
			continue
		}
		if !confs[name] {
			o.WgConfigs = append(o.WgConfigs, filepath.Join(wgDir, name))
		}
	}

	sort.Strings(o.SSHKeys)
	sort.Strings(o.Secrets)
	sort.Strings(o.WgConfigs)
	return o, nil
}

// RemoveInstanceRecords deletes the given instance IDs from the registry while
// preserving every other entry and field, writing back atomically.
func RemoveInstanceRecords(instanceFile string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	var all map[string]map[string]interface{}
	if _, err := toml.DecodeFile(instanceFile, &all); err != nil {
		return fmt.Errorf("decode %s: %w", instanceFile, err)
	}
	for _, id := range ids {
		delete(all, id)
	}

	tmp := instanceFile + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := toml.NewEncoder(f).Encode(all); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, instanceFile)
}

// listFiles returns the names of regular files directly under dir. A missing
// directory yields no files rather than an error.
func listFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	return names
}
