package utils

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

// InstanceRecord is the provider-agnostic subset of a registry entry. All three
// providers write these same toml tags, so the registry can be read uniformly
// regardless of which provider created the instance.
type InstanceRecord struct {
	Id          string `toml:"id"`
	PrivKeyPath string `toml:"priv_key_path"`
	Provider    string `toml:"provider"`
	VPSName     string `toml:"vps_name"`
}

type InstanceDB map[string]InstanceRecord

func LoadInstances(path string) (InstanceDB, error) {
	db := InstanceDB{}
	if _, err := toml.DecodeFile(path, &db); err != nil {
		return nil, fmt.Errorf("decode instances.toml: %w", err)
	}
	return db, nil
}

// instanceID is the provider ID, e.g., "530066077"
func GetVPSNameForInstance(path, instanceID string) (string, error) {
	db, err := LoadInstances(path)
	if err != nil {
		return "", fmt.Errorf("load instances from %s: %w", path, err)
	}

	rec, ok := db[instanceID]
	if !ok {
		return "", fmt.Errorf("instance %s not found in %s", instanceID, path)
	}
	if rec.VPSName == "" {
		return "", fmt.Errorf("instance %s has empty vps_name in %s", instanceID, path)
	}

	return rec.VPSName, nil
}
