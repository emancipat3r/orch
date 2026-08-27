package utils

import (
	"path/filepath"
	"testing"
)

func TestPathsEnsureAndHelpers(t *testing.T) {
	root := filepath.Join(t.TempDir(), "orch")
	p := PathsUnder(root)

	var created []string
	if err := p.Ensure(func(d string) { created = append(created, d) }); err != nil {
		t.Fatal(err)
	}
	if len(created) != len(p.Dirs()) {
		t.Fatalf("expected %d dirs created, got %d", len(p.Dirs()), len(created))
	}
	for _, d := range p.Dirs() {
		if !DirExists(d) {
			t.Errorf("missing dir %s", d)
		}
	}
	// Second call is a no-op.
	created = nil
	if err := p.Ensure(func(d string) { created = append(created, d) }); err != nil {
		t.Fatal(err)
	}
	if len(created) != 0 {
		t.Errorf("Ensure re-created %v", created)
	}

	if got, want := p.WgConf("vps", "1.2.3.4"), filepath.Join(root, "wg", "vps.conf"); got != want {
		t.Errorf("WgConf name: %s != %s", got, want)
	}
	if got, want := p.WgConf("", "1.2.3.4"), filepath.Join(root, "wg", "client-1.2.3.4.conf"); got != want {
		t.Errorf("WgConf legacy: %s != %s", got, want)
	}
	if p.WgConf("", "pending") != "" || p.WgConf("", "") != "" {
		t.Error("WgConf should be empty without a usable name or ip")
	}
	if got, want := p.PassFile(filepath.Join(root, ".ssh", "do-abc")), filepath.Join(root, "secrets", "do-abc.pass"); got != want {
		t.Errorf("PassFile: %s != %s", got, want)
	}
}
