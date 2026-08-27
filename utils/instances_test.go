package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

type fakeLinode struct {
	Id       string `toml:"id"`
	HostUUID string `toml:"host_uuid"`
	SSHKeyID string `toml:"ssh_key_id"`
	Provider string `toml:"provider"`
}

type fakeDO struct {
	Id       string `toml:"id"`
	KeyID    string `toml:"ssh_key_id"`
	Provider string `toml:"provider"`
}

func TestUpsertPreservesOtherProvidersFields(t *testing.T) {
	f := filepath.Join(t.TempDir(), "instances.toml")

	// Registry doesn't exist yet: first upsert creates it.
	if err := UpsertInstanceRecord(f, "do-1", fakeDO{Id: "do-1", KeyID: "42", Provider: "DigitalOcean"}); err != nil {
		t.Fatal(err)
	}
	// A different provider with a different struct shape writes next.
	if err := UpsertInstanceRecord(f, "lin-1", fakeLinode{Id: "lin-1", HostUUID: "uuid", SSHKeyID: "7", Provider: "Linode"}); err != nil {
		t.Fatal(err)
	}

	var raw map[string]map[string]interface{}
	if _, err := toml.DecodeFile(f, &raw); err != nil {
		t.Fatal(err)
	}
	if raw["do-1"]["ssh_key_id"] != "42" {
		t.Errorf("DO ssh_key_id lost after Linode write: %v", raw["do-1"])
	}
	if raw["lin-1"]["host_uuid"] != "uuid" {
		t.Errorf("Linode host_uuid missing: %v", raw["lin-1"])
	}

	// Replacing an entry keeps everything else intact.
	if err := UpsertInstanceRecord(f, "do-1", fakeDO{Id: "do-1", KeyID: "43", Provider: "DigitalOcean"}); err != nil {
		t.Fatal(err)
	}
	db, err := LoadInstances(f)
	if err != nil {
		t.Fatal(err)
	}
	if db["do-1"].SSHKeyID != "43" || db["lin-1"].SSHKeyID != "7" {
		t.Errorf("unexpected db: %+v", db)
	}
	if _, err := os.Stat(f + ".tmp"); !os.IsNotExist(err) {
		t.Error("tmp file left behind")
	}
}

func TestLoadInstancesLegacyKeyIDAndMissingFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "instances.toml")
	db, err := LoadInstances(f)
	if err != nil || len(db) != 0 {
		t.Fatalf("missing file should be empty registry, got %v %v", db, err)
	}

	conf := "[555]\nid = \"555\"\nkey_id = \"99\"\nipv4 = \"1.2.3.4\"\nprovider = \"DigitalOcean\"\nvps_name = \"vps\"\n"
	if err := os.WriteFile(f, []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}
	db, err = LoadInstances(f)
	if err != nil {
		t.Fatal(err)
	}
	rec := db["555"]
	if rec.SSHKeyID != "99" || rec.Ipv4 != "1.2.3.4" || rec.VPSName != "vps" {
		t.Errorf("legacy record not mapped: %+v", rec)
	}
}

func TestCleanupLocalArtifacts(t *testing.T) {
	p := PathsUnder(t.TempDir())
	if err := p.Ensure(nil); err != nil {
		t.Fatal(err)
	}
	priv := filepath.Join(p.SSH, "do-abc")
	for _, f := range []string{priv, priv + ".pub", p.PassFile(priv), p.WgConf("vps", ""), p.WgConf("", "1.2.3.4")} {
		if err := os.WriteFile(f, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// VPSName deliberately empty so the netns teardown (needs root) is skipped.
	errs := CleanupLocalArtifacts(p, InstanceRecord{PrivKeyPath: priv, Ipv4: "1.2.3.4"})
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	for _, f := range []string{priv, priv + ".pub", p.PassFile(priv), p.WgConf("", "1.2.3.4")} {
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Errorf("%s should have been removed", f)
		}
	}
	if _, err := os.Stat(p.WgConf("vps", "")); err != nil {
		t.Errorf("named config should be untouched when VPSName is empty")
	}
	// Second run over already-deleted files is silent.
	if errs := CleanupLocalArtifacts(p, InstanceRecord{PrivKeyPath: priv, Ipv4: "1.2.3.4"}); len(errs) != 0 {
		t.Errorf("repeat cleanup errored: %v", errs)
	}
}
