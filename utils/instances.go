package utils

import (
	"bytes"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// InstanceRecord is the provider-agnostic subset of a registry entry: the
// fields every provider writes and that destroy/prune need for cleanup. The
// registry file itself is handled as raw TOML tables (see UpsertInstanceRecord)
// so that provider-specific fields are never lost when another provider
// rewrites the file.
type InstanceRecord struct {
	Id          string `toml:"id"`
	Ipv4        string `toml:"ipv4"`
	Label       string `toml:"label"`
	PrivKeyPath string `toml:"priv_key_path"`
	Provider    string `toml:"provider"`
	SSHKeyID    string `toml:"ssh_key_id"`
	VPSName     string `toml:"vps_name"`
}

type InstanceDB map[string]InstanceRecord

// rawRegistry is the registry as untyped TOML tables keyed by instance id.
type rawRegistry map[string]map[string]interface{}

// loadRaw reads the registry without imposing any struct shape. A missing
// file is an empty registry.
func loadRaw(path string) (rawRegistry, error) {
	all := rawRegistry{}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return all, nil
		}
		return nil, err
	}
	if _, err := toml.DecodeFile(path, &all); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return all, nil
}

// writeRaw writes the registry atomically (tmp + rename) so a crash mid-write
// can never leave a truncated file behind.
func writeRaw(path string, all rawRegistry) error {
	tmp := path + ".tmp"
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
	return os.Rename(tmp, path)
}

func str(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// LoadInstances returns the provider-agnostic view of every registry entry.
// A missing file yields an empty DB. The legacy DigitalOcean "key_id" field is
// accepted as an alias for ssh_key_id.
func LoadInstances(path string) (InstanceDB, error) {
	all, err := loadRaw(path)
	if err != nil {
		return nil, err
	}
	db := InstanceDB{}
	for id, m := range all {
		rec := InstanceRecord{
			Id:          str(m, "id"),
			Ipv4:        str(m, "ipv4"),
			Label:       str(m, "label"),
			PrivKeyPath: str(m, "priv_key_path"),
			Provider:    str(m, "provider"),
			SSHKeyID:    str(m, "ssh_key_id"),
			VPSName:     str(m, "vps_name"),
		}
		if rec.SSHKeyID == "" {
			rec.SSHKeyID = str(m, "key_id")
		}
		if rec.Id == "" {
			rec.Id = id
		}
		db[id] = rec
	}
	return db, nil
}

// UpsertInstanceRecord adds or replaces the entry for id with rec (any struct
// with toml tags) while leaving every other entry byte-for-byte intact. This
// is the only way providers should write to the registry: decoding the whole
// file into one provider's struct and re-encoding it silently renames or drops
// the other providers' fields.
func UpsertInstanceRecord(path, id string, rec interface{}) error {
	all, err := loadRaw(path)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(rec); err != nil {
		return fmt.Errorf("encode record: %w", err)
	}
	var m map[string]interface{}
	if _, err := toml.Decode(buf.String(), &m); err != nil {
		return fmt.Errorf("re-decode record: %w", err)
	}
	all[id] = m
	return writeRaw(path, all)
}

// RemoveInstanceRecords deletes the given instance IDs from the registry while
// preserving every other entry and field. A missing file is a no-op.
func RemoveInstanceRecords(path string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	all, err := loadRaw(path)
	if err != nil {
		return err
	}
	if len(all) == 0 {
		return nil
	}
	for _, id := range ids {
		delete(all, id)
	}
	return writeRaw(path, all)
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
