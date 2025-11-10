package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

func AssignIP(interfaceName, ipWithCIDR string) error {
	link, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return logger.Errorf("failed to get interface %s: %v", interfaceName, err)
	}
	addr, err := netlink.ParseAddr(ipWithCIDR)
	if err != nil {
		return logger.Errorf("failed to parse IP address %s: %v", ipWithCIDR, err)
	}
	// Replace avoids EEXIST if you re-run
	if err := netlink.AddrReplace(link, addr); err != nil {
		return logger.Errorf("failed to assign %s to %s: %v", ipWithCIDR, interfaceName, err)
	}
	// Optional: ensure link is up
	if err := netlink.LinkSetUp(link); err != nil {
		return logger.Errorf("failed to set %s up: %v", interfaceName, err)
	}
	logger.Info("IP address " + ipWithCIDR + " assigned to interface " + interfaceName)
	return nil
}

func SetWgConf(interfaceName, wgConfPath string) error {
	data, err := os.ReadFile(wgConfPath)
	if err != nil {
		logger.Error("Failed to read WireGuard config: " + logger.Highlight(wgConfPath))
		return err
	}

	config, err := wgtypes.ParseConfig(string(data))
	parsec
	if err != nil {
		logger.Error("Failed to parse WireGuard config: " + logger.Highlight(wgConfPath))
		return err
	}

	client, err := wgctrl.New()
	if err != nil {
		logger.Error("Failed to open wgctrl client")
		return err
	}
	defer client.Close()

	if err := client.ConfigureDevice(interfaceName, *config); err != nil {
		return logger.Errorf("failed to configure %s: %v", interfaceName, err)
	}
	logger.Info("Applied WireGuard config to " + interfaceName)
	return nil
}

func CreateNetNS(netNsName string) (netns.NsHandle, error) {
	nsPath := filepath.Join("/var/run/netns", netNsName)

	if err := os.MkdirAll("/var/run/netns", 0o755); err != nil {
		return -1, fmt.Errorf("failed to ensure /var/run/netns: %w", err)
	}

	newNs, err := netns.NewNamed(netNsName)
	if err != nil {
		return -1, fmt.Errorf("failed to create netns %q: %w", netNsName, err)
	}

	logger.Info("Created network namespace: " + logger.Highlight(netNsName) + " (" + nsPath + ")")
	return newNs, nil
}

// MoveIntToNS moves an interface (e.g., "wg0") into nsHandle and brings it up there.
func MoveIntToNS(nsHandle netns.NsHandle, interfaceName string) error {
	// Lookup in current ns
	link, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return logger.Errorf("lookup %q in source ns: %v", interfaceName, err)
	}

	_ = netlink.LinkSetDown(link)

	if err := netlink.LinkSetNsFd(link, int(nsHandle)); err != nil {
		return logger.Errorf("move %q to target ns: %v", interfaceName, err)
	}

	// Work inside target ns without changing process-wide ns
	h, err := netlink.NewHandleAt(nsHandle)
	if err != nil {
		return logger.Errorf("open netlink handle in target ns: %v", err)
	}
	defer h.Close()

	moved, err := h.LinkByName(interfaceName)
	if err != nil {
		return logger.Errorf("find %q in target ns after move: %v", interfaceName, err)
	}
	if err := h.LinkSetUp(moved); err != nil {
		return logger.Errorf("set %q up in target ns: %v", interfaceName, err)
	}

	logger.Info("Moved " + logger.Highlight(interfaceName) + " to target ns and set it UP")
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
