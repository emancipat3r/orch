package utils

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/emancipat3r/vps3/logger"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"golang.org/x/sys/unix"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func CreateWgInt(interfaceName string) (string, error) {
	la := netlink.NewLinkAttrs()
	la.Name = interfaceName
	if err := netlink.LinkAdd(&netlink.Wireguard{LinkAttrs: la}); err != nil {
		return "", logger.Errorf("failed to create WireGuard interface %s: %v", interfaceName, err)
	}
	logger.Info("WireGuard interface " + interfaceName + " created successfully")
	return interfaceName, nil
}

func ParseConfigForIP(configPath string) (string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		// be tolerant of case and leading key spacing
		if strings.HasPrefix(strings.ToLower(line), "address =") {
			addr := strings.TrimSpace(strings.TrimPrefix(line, "Address ="))
			if addr != "" {
				// NOTE: WireGuard allows multiple addresses, comma-separated.
				// If present, take the first.
				if idx := strings.Index(addr, ","); idx >= 0 {
					addr = strings.TrimSpace(addr[:idx])
				}
				return addr, nil
			}
		}
	}
	return "", errors.New("address not found in WireGuard config")
}

// SetWgConf emulates `wg setconf <iface> <file>` in Go.
func SetWgConf(interfaceName, wgConfPath string) error {
	data, err := os.ReadFile(wgConfPath)
	if err != nil {
		return logger.Errorf("read %s: %v", wgConfPath, err)
	}

	cfg, err := parseWGConf(data)
	if err != nil {
		return logger.Errorf("parse %s: %v", wgConfPath, err)
	}

	client, err := wgctrl.New()
	if err != nil {
		return logger.Errorf("open wgctrl: %v", err)
	}
	defer client.Close()

	if err := client.ConfigureDevice(interfaceName, *cfg); err != nil {
		return logger.Errorf("Failed to configure %s: %v", interfaceName, err)
	}

	logger.Info("Applied WireGuard config to " + logger.Highlight(interfaceName))
	return nil

}

// parseWGConf is a minimal parser for wg-quick style config files.
// It returns a wgtypes.Config equivalent to what `wg setconf` would apply.
func parseWGConf(data []byte) (*wgtypes.Config, error) {
	sc := bufio.NewScanner(bytes.NewReader(data))
	cfg := &wgtypes.Config{ReplacePeers: true}

	var section string
	var peer *wgtypes.PeerConfig
	commitPeer := func() {
		if peer != nil {
			cfg.Peers = append(cfg.Peers, *peer)
			peer = nil
		}
	}

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if section == "Peer" {
				commitPeer()
			}
			section = strings.Trim(line, "[]")
			if section == "Peer" {
				peer = &wgtypes.PeerConfig{}
			}
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch section {
		case "Interface":
			switch strings.ToLower(key) {
			case "privatekey":
				k, err := wgtypes.ParseKey(val)
				if err != nil {
					return nil, fmt.Errorf("invalid private key: %w", err)
				}
				cfg.PrivateKey = &k
			case "listenport":
				p, err := strconv.Atoi(val)
				if err == nil && p > 0 && p < 65536 {
					cfg.ListenPort = &p
				}
			}
		case "Peer":
			switch strings.ToLower(key) {
			case "publickey":
				k, err := wgtypes.ParseKey(val)
				if err != nil {
					return nil, fmt.Errorf("invalid peer pubkey: %w", err)
				}
				peer.PublicKey = k
			case "presharedkey":
				k, err := wgtypes.ParseKey(val)
				if err != nil {
					return nil, fmt.Errorf("invalid preshared key: %w", err)
				}
				peer.PresharedKey = &k
			case "endpoint":
				udp, err := net.ResolveUDPAddr("udp", val) // <-- fix
				if err != nil {
					return nil, fmt.Errorf("invalid endpoint %q: %w", val, err)
				}
				peer.Endpoint = udp
			case "allowedips":
				peer.AllowedIPs = nil
				for _, s := range strings.Split(val, ",") {
					s = strings.TrimSpace(s)
					if s == "" {
						continue
					}
					_, ipnet, err := net.ParseCIDR(s)
					if err != nil {
						return nil, fmt.Errorf("invalid allowedip %q: %w", s, err)
					}
					peer.AllowedIPs = append(peer.AllowedIPs, *ipnet)
				}
			case "persistentkeepalive":
				secs, _ := strconv.Atoi(val)
				if secs > 0 {
					d := time.Duration(secs) * time.Second
					peer.PersistentKeepaliveInterval = &d
				}
			}
		}
	}
	if section == "Peer" {
		commitPeer()
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func CreateNetNS(netNsName string) (netns.NsHandle, error) {
	nsPath := filepath.Join("/var/run/netns", netNsName)

	if err := os.MkdirAll("/var/run/netns", 0o755); err != nil {
		return -1, fmt.Errorf("failed to ensure /var/run/netns: %w", err)
	}

	// Save current namespace
	orig, err := netns.Get()
	if err != nil {
		return -1, fmt.Errorf("failed to get current netns: %w", err)
	}
	defer orig.Close()

	// Create named namespace (this likely moves us into the new one)
	newNs, err := netns.NewNamed(netNsName)
	if err != nil {
		return -1, fmt.Errorf("failed to create netns %q: %w", netNsName, err)
	}

	// Restore original ns so future calls (like LinkByName) see host interfaces
	if err := netns.Set(orig); err != nil {
		newNs.Close()
		return -1, fmt.Errorf("failed to restore original netns: %w", err)
	}

	logger.Info("Created network namespace: " + logger.Highlight(netNsName) + " (" + nsPath + ")")
	return newNs, nil
}

// MoveIntToNS moves an interface into specified netns and brings it up in there
func MoveIntToNS(nsHandle netns.NsHandle, linkName string) error {
	// Lookup in current (host) ns
	link, err := netlink.LinkByName(linkName)
	if err != nil {
		return logger.Errorf("Failed lookup of interface %q in host netns: %v", linkName, err)
	}

	_ = netlink.LinkSetDown(link)

	if err := netlink.LinkSetNsFd(link, int(nsHandle)); err != nil {
		return logger.Errorf("Moved %q to target ns: %v", linkName, err)
	}

	// Work inside target ns without changing process-wide ns
	h, err := netlink.NewHandleAt(nsHandle)
	if err != nil {
		return logger.Errorf("open netlink handle in target ns: %v", err)
	}
	defer h.Close()

	moved, err := h.LinkByName(linkName)
	if err != nil {
		return logger.Errorf("find %q in target ns after move: %v", linkName, err)
	}
	if err := h.LinkSetUp(moved); err != nil {
		return logger.Errorf("set %q up in target ns: %v", linkName, err)
	}

	logger.Info("Moved " + logger.Highlight(linkName) + " to target ns and set it UP")
	return nil
}

// TearDownNamespace deletes a named netns like `ip netns del <name>`.
// Close our handle, attempt delete, retry if busy, lazy-unmount as last resort.
func TearDownNamespace(netNsName string, h netns.NsHandle) error {
	nsPath := filepath.Join("/var/run/netns", netNsName)

	// Drop our reference if any
	if int(h) > 0 {
		_ = h.Close()
	}

	// Treat ENOENT as success (already gone)
	tryDelete := func() error {
		err := netns.DeleteNamed(netNsName)
		if err == nil {
			return nil
		}
		// Already removed
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, unix.ENOENT) {
			return nil
		}
		return err
	}

	if err := tryDelete(); err != nil {
		if isBusy(err) {
			for i := 0; i < 3; i++ {
				time.Sleep(150 * time.Millisecond)
				if err2 := tryDelete(); err2 == nil {
					logger.Info("Namespace " + logger.Highlight(netNsName) + " deleted")
					return nil
				} else if !isBusy(err2) {
					return logger.Errorf("delete netns %q: %v", netNsName, err2)
				}
			}
			// Last resort: lazy unmount
			_ = unix.Unmount(nsPath, unix.MNT_DETACH)
			if _, statErr := os.Stat(nsPath); statErr == nil {
				return logger.Errorf("namespace %q still mounted/busy at %s", netNsName, nsPath)
			}
			logger.Info("Namespace " + logger.Highlight(netNsName) + " deleted (lazy unmount)")
			return nil
		}
		return logger.Errorf("delete netns %q: %v", netNsName, err)
	}

	logger.Info("Namespace " + logger.Highlight(netNsName) + " deleted")
	return nil
}

func isBusy(err error) bool {
	return errors.Is(err, unix.EBUSY)
}

func ParseDNSFromConfig(configPath string) ([]string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "dns =") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "DNS ="))
			if val == "" {
				continue
			}
			var servers []string
			for _, part := range strings.Split(val, ",") {
				s := strings.TrimSpace(part)
				if s != "" {
					servers = append(servers, s)
				}
			}
			if len(servers) == 0 {
				return nil, errors.New("DNS line present but no servers parsed")
			}
			return servers, nil
		}
	}
	return nil, errors.New("DNS not found in WireGuard config")
}

func EnsureNetnsResolvConf(netnsName string, nameservers []string) error {
	if len(nameservers) == 0 {
		nameservers = []string{"1.1.1.1"}
	}

	dir := filepath.Join("/etc/netns", netnsName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return logger.Errorf("create /etc/netns dir: %v", err)
	}

	resolvPath := filepath.Join(dir, "resolv.conf")

	var sb strings.Builder
	for _, ns := range nameservers {
		sb.WriteString("nameserver ")
		sb.WriteString(ns)
		sb.WriteString("\n")
	}

	if err := os.WriteFile(resolvPath, []byte(sb.String()), 0o644); err != nil {
		return logger.Errorf("write %s: %v", resolvPath, err)
	}

	logger.Info("Wrote per-netns resolv.conf for " +
		logger.Highlight(netnsName) + " at " + logger.Highlight(resolvPath))
	return nil
}
