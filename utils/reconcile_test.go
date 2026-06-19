package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("touch %s: %v", path, err)
	}
}

func TestFindOrphans(t *testing.T) {
	root := t.TempDir()
	sshDir := filepath.Join(root, ".ssh")
	secretsDir := filepath.Join(root, "secrets")
	wgDir := filepath.Join(root, "wg")
	for _, d := range []string{sshDir, secretsDir, wgDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Referenced by the registry — must NOT be reported as orphans.
	keep := filepath.Join(sshDir, "linode-111")
	touch(t, keep)
	touch(t, keep+".pub")
	touch(t, filepath.Join(secretsDir, "linode-111.pass"))
	touch(t, filepath.Join(wgDir, "vps-a.conf"))

	// Unreferenced leftovers — must be reported.
	orphKey := filepath.Join(sshDir, "linode-999")
	touch(t, orphKey)
	touch(t, orphKey+".pub")
	touch(t, filepath.Join(secretsDir, "linode-999.pass"))
	touch(t, filepath.Join(wgDir, "vps-z.conf"))

	instFile := filepath.Join(root, "instances.toml")
	conf := "[111]\nid = \"111\"\npriv_key_path = \"" + keep + "\"\nvps_name = \"vps-a\"\nprovider = \"Linode\"\n"
	if err := os.WriteFile(instFile, []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}

	o, err := FindOrphans(instFile, sshDir, secretsDir, wgDir)
	if err != nil {
		t.Fatalf("FindOrphans: %v", err)
	}

	if len(o.SSHKeys) != 1 || o.SSHKeys[0] != orphKey {
		t.Errorf("ssh orphans = %v, want [%s]", o.SSHKeys, orphKey)
	}
	if len(o.Secrets) != 1 {
		t.Errorf("secret orphans = %v, want 1", o.Secrets)
	}
	if len(o.WgConfigs) != 1 {
		t.Errorf("wg orphans = %v, want 1", o.WgConfigs)
	}
	if o.Total() != 3 {
		t.Errorf("total = %d, want 3", o.Total())
	}
}

func TestFindOrphansEmptyRegistry(t *testing.T) {
	root := t.TempDir()
	sshDir := filepath.Join(root, ".ssh")
	if err := os.MkdirAll(sshDir, 0o755); err != nil {
		t.Fatal(err)
	}
	touch(t, filepath.Join(sshDir, "stray-key"))

	// No registry file at all: everything on disk is orphaned.
	o, err := FindOrphans(filepath.Join(root, "instances.toml"), sshDir, root, root)
	if err != nil {
		t.Fatalf("FindOrphans: %v", err)
	}
	if len(o.SSHKeys) != 1 {
		t.Errorf("ssh orphans = %v, want 1", o.SSHKeys)
	}
}

func TestRemoveInstanceRecords(t *testing.T) {
	instFile := filepath.Join(t.TempDir(), "instances.toml")
	conf := "[111]\nid = \"111\"\nprovider = \"Linode\"\n\n[222]\nid = \"222\"\nprovider = \"Vultr\"\n"
	if err := os.WriteFile(instFile, []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := RemoveInstanceRecords(instFile, []string{"111"}); err != nil {
		t.Fatalf("RemoveInstanceRecords: %v", err)
	}

	db, err := LoadInstances(instFile)
	if err != nil {
		t.Fatalf("LoadInstances: %v", err)
	}
	if _, ok := db["111"]; ok {
		t.Errorf("111 should have been removed")
	}
	if _, ok := db["222"]; !ok {
		t.Errorf("222 should have been kept")
	}
}
