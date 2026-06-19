package utils

import (
	"os"
	"path/filepath"
	"testing"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func writeTempConf(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "wg.conf")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp conf: %v", err)
	}
	return p
}

func TestParseWGConf(t *testing.T) {
	priv, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Fatalf("gen private key: %v", err)
	}
	peer, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Fatalf("gen peer key: %v", err)
	}
	psk, err := wgtypes.GenerateKey()
	if err != nil {
		t.Fatalf("gen psk: %v", err)
	}

	conf := "[Interface]\n" +
		"PrivateKey = " + priv.String() + "\n" +
		"Address = 10.0.0.2/32\n" +
		"ListenPort = 51820\n" +
		"DNS = 1.1.1.1\n\n" +
		"[Peer]\n" +
		"PublicKey = " + peer.PublicKey().String() + "\n" +
		"PresharedKey = " + psk.String() + "\n" +
		"Endpoint = 192.0.2.1:51820\n" +
		"AllowedIPs = 0.0.0.0/0, ::/0\n" +
		"PersistentKeepalive = 25\n"

	cfg, err := parseWGConf([]byte(conf))
	if err != nil {
		t.Fatalf("parseWGConf: %v", err)
	}

	if cfg.PrivateKey == nil || cfg.PrivateKey.String() != priv.String() {
		t.Errorf("private key mismatch")
	}
	if cfg.ListenPort == nil || *cfg.ListenPort != 51820 {
		t.Errorf("listen port = %v, want 51820", cfg.ListenPort)
	}
	if len(cfg.Peers) != 1 {
		t.Fatalf("peers = %d, want 1", len(cfg.Peers))
	}

	pe := cfg.Peers[0]
	if pe.PublicKey.String() != peer.PublicKey().String() {
		t.Errorf("peer public key mismatch")
	}
	if pe.Endpoint == nil || pe.Endpoint.String() != "192.0.2.1:51820" {
		t.Errorf("endpoint = %v, want 192.0.2.1:51820", pe.Endpoint)
	}
	if len(pe.AllowedIPs) != 2 {
		t.Errorf("allowed ips = %d, want 2", len(pe.AllowedIPs))
	}
	if pe.PersistentKeepaliveInterval == nil || pe.PersistentKeepaliveInterval.Seconds() != 25 {
		t.Errorf("keepalive = %v, want 25s", pe.PersistentKeepaliveInterval)
	}
}

func TestParseConfigForIP(t *testing.T) {
	// Multiple comma-separated addresses: the first one wins.
	p := writeTempConf(t, "[Interface]\nAddress = 10.0.0.2/32, fd00::2/128\n")
	ip, err := ParseConfigForIP(p)
	if err != nil {
		t.Fatalf("ParseConfigForIP: %v", err)
	}
	if ip != "10.0.0.2/32" {
		t.Errorf("ip = %q, want 10.0.0.2/32", ip)
	}

	if _, err := ParseConfigForIP(writeTempConf(t, "[Interface]\nPrivateKey = x\n")); err == nil {
		t.Errorf("expected error when Address is absent")
	}
}

func TestParseDNSFromConfig(t *testing.T) {
	p := writeTempConf(t, "[Interface]\nDNS = 1.1.1.1, 8.8.8.8\n")
	dns, err := ParseDNSFromConfig(p)
	if err != nil {
		t.Fatalf("ParseDNSFromConfig: %v", err)
	}
	if len(dns) != 2 || dns[0] != "1.1.1.1" || dns[1] != "8.8.8.8" {
		t.Errorf("dns = %v, want [1.1.1.1 8.8.8.8]", dns)
	}

	if _, err := ParseDNSFromConfig(writeTempConf(t, "[Interface]\nAddress = 10.0.0.2/32\n")); err == nil {
		t.Errorf("expected error when DNS is absent")
	}
}

func TestParseCreds(t *testing.T) {
	p := filepath.Join(t.TempDir(), "configuration.toml")
	conf := "[linode]\nkey = \"abc123\"\n\n[digitalocean]\nkey = \"\"\n\n[vultr]\nkey = \"vk\"\n"
	if err := os.WriteFile(p, []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}

	// Provider name matches ui.ChoiceProvider output; ParseCreds lowercases it.
	if got := ParseCreds(p, "Linode"); got != "abc123" {
		t.Errorf("Linode key = %q, want abc123", got)
	}
	if got := ParseCreds(p, "Vultr"); got != "vk" {
		t.Errorf("Vultr key = %q, want vk", got)
	}
	if got := ParseCreds(p, "DigitalOcean"); got != "" {
		t.Errorf("DigitalOcean key = %q, want empty", got)
	}
	if got := ParseCreds(p, "Nope"); got != "" {
		t.Errorf("unknown provider key = %q, want empty", got)
	}
}
