package cmd

import (
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/keygen"
	"github.com/emancipat3r/orch/logger"
	"github.com/emancipat3r/orch/providers"
	"github.com/emancipat3r/orch/ui"
	"github.com/emancipat3r/orch/utils"
	"github.com/spf13/cobra"
	"github.com/vishvananda/netlink"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Provision a new VPS instance",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		// Preflight: confirm the local kernel can build the WireGuard tunnel
		// before spending money provisioning a remote VPS.
		if err := utils.CheckWireGuardSupport(); err != nil {
			logger.Error("WireGuard preflight check failed: " + err.Error())
			return
		}

		providerName := ui.ChoiceProvider()
		if providerName == "" {
			logger.Info("Operation cancelled by user.")
			return
		}
		if ctx.Err() != nil {
			return
		}
		logger.Info("You selected: " + logger.Highlight(providerName))

		prov, err := providers.GetProvider(providerName, configFile)
		if err != nil {
			logger.Error(err.Error())
			return
		}

		balance, err := prov.Balance(ctx)
		if err != nil {
			logger.Error("Failed to get " + providerName + " account balance: " + err.Error())
			return
		}
		logger.Info(providerName + " account balance: " + logger.Highlight("$"+balance))

		// Load all selectable options up front so the wizard is fully navigable.
		regions, err := prov.Regions(ctx)
		if err != nil {
			logger.Error("Failed to load regions: " + err.Error())
			return
		}
		images, err := prov.Images(ctx)
		if err != nil {
			logger.Error("Failed to load images: " + err.Error())
			return
		}
		sizes, err := prov.Sizes(ctx)
		if err != nil {
			logger.Error("Failed to load sizes: " + err.Error())
			return
		}

		regionLabels, regionValue := optionLabels(regions)
		imageLabels, imageValue := optionLabels(images)
		sizeLabels, sizeValue := optionLabels(sizes)

		var region, image, size string
		steps := []ui.WizardStep{
			{Key: "region", Title: "Select your region", Options: regionLabels, Value: &region},
			{Key: "image", Title: "Select your image", Options: imageLabels, Value: &image},
			{Key: "size", Title: "Select your resourcing", Options: sizeLabels, Value: &size},
		}

		selections, err := ui.CreateNavigableWizard(steps)
		if err != nil {
			logger.Info("Operation cancelled by user.")
			return
		}

		selectedRegion := selections["region"]
		selectedImage := selections["image"]
		selectedSize := selections["size"]

		logger.Info("You selected region: " + logger.Highlight(selectedRegion))
		logger.Info("You selected image: " + logger.Highlight(selectedImage))
		logger.Info("You selected type: " + logger.Highlight(selectedSize))

		// === Per-instance keypair (using charmbracelet/keygen) ===
		prefix := providerPrefix(providerName)
		keyName := prefix + "-" + providers.CreateUID()
		perPriv := pathSSH + keyName // ~/.config/orch/.ssh/<provider>-XXXXXXXX
		perPub := perPriv + ".pub"

		pass, err := utils.GenerateRandomPassword(24)
		if err != nil {
			logger.Error("Failed to gen passphrase: " + err.Error())
			return
		}
		if err := os.WriteFile(pathSecrets+keyName+".pass", []byte(pass+"\n"), 0600); err != nil {
			logger.Error("Failed to store passphrase: " + err.Error())
			return
		}

		if _, err := keygen.New(perPriv, keygen.WithKeyType(keygen.Ed25519), keygen.WithPassphrase(pass), keygen.WithWrite()); err != nil {
			logger.Error("Failed to create per-instance keypair: " + err.Error())
			return
		}

		if err := utils.AddKeyToSSHAgent(perPriv); err != nil {
			logger.Debug("Could not add key to ssh-agent: " + err.Error())
		}

		sshKeyID, err := prov.UploadSSHKey(ctx, perPub)
		if err != nil {
			logger.Error("Failed to upload SSH key: " + err.Error())
			return
		}
		logger.Info("Per-instance SSH key ID: " + logger.Highlight(sshKeyID))

		if vpsName == "" {
			vpsName = prefix + "-" + providers.CreateUID()
		}

		logger.Info("Creating " + providerName + " instance...")
		inst, err := prov.Create(ctx, providers.CreateSpec{
			Image:        imageValue[selectedImage],
			Region:       regionValue[selectedRegion],
			Size:         sizeValue[selectedSize],
			SSHKeyID:     sshKeyID,
			PrivKeyPath:  perPriv,
			VPSName:      vpsName,
			InstanceFile: instanceFile,
		})
		if err != nil {
			logger.Error("Failed to create " + providerName + " instance: " + err.Error())
			// No instance came up, so the local keypair/passphrase we generated
			// for it are orphans — clean them up rather than leaving artifacts.
			if rmErr := utils.RemoveKeyFromSSHAgent(perPriv); rmErr != nil {
				logger.Debug("Could not remove key from ssh-agent: " + rmErr.Error())
			}
			_ = os.Remove(perPriv)
			_ = os.Remove(perPub)
			_ = os.Remove(pathSecrets + keyName + ".pass")
			return
		}

		if err := setupWireGuard(vpsName, vpsName); err != nil {
			logger.Error("WireGuard setup failed: " + err.Error())
			// The remote VPS is provisioned and billing but has no working
			// tunnel. Offer to tear it down so a failed setup doesn't silently
			// cost money.
			if inst.ID != "" && ui.Confirm("WireGuard setup failed. Destroy the just-created "+providerName+" instance "+inst.ID+" to avoid charges?") {
				if derr := prov.Destroy(ctx, inst.ID, instanceFile); derr != nil {
					logger.Error("Teardown failed: " + derr.Error())
				} else {
					logger.Info("Destroyed instance " + logger.Highlight(inst.ID))
				}
			}
			return
		}
		logger.Info("WireGuard network namespace setup complete.")
	},
}

// optionLabels splits provider Options into the label slice the wizard renders
// and a label->value lookup so the selected display string maps back to the
// underlying id/slug without positional string parsing.
func optionLabels(opts []providers.Option) ([]string, map[string]string) {
	labels := make([]string, len(opts))
	lookup := make(map[string]string, len(opts))
	for i, o := range opts {
		labels[i] = o.Label
		lookup[o.Label] = o.Value
	}
	return labels, lookup
}

// providerPrefix is the short, filesystem-friendly key prefix for a provider's
// per-instance SSH keys and default VPS names.
func providerPrefix(name string) string {
	switch name {
	case "DigitalOcean":
		return "do"
	case "Linode":
		return "linode"
	case "Vultr":
		return "vultr"
	default:
		return strings.ToLower(name)
	}
}

func setupWireGuard(vpsName, netnsName string) (err error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return logger.Errorf("failed to get home directory: %v", err)
	}

	confPath := filepath.Join(homeDir, ".config/orch/wg", vpsName+".conf")
	if _, err := os.Stat(confPath); err != nil {
		return logger.Errorf("WireGuard config not found at %s", confPath)
	}

	if netnsName == "" {
		netnsName = vpsName
	}

	linkName := "wg-" + netnsName
	logger.Info("Setting up WireGuard: " + logger.Highlight(netnsName))

	// 1) Create WG link in host ns
	if _, err := utils.CreateWgInt(linkName); err != nil {
		return err
	}

	// From here on, any failure must not leak the interface or namespace we
	// created. Roll back whatever exists if we don't reach the end.
	success := false
	defer func() {
		if !success {
			logger.Warn("Rolling back partial WireGuard setup for " + logger.Highlight(netnsName))
			utils.CleanupWireGuard(linkName, netnsName)
		}
	}()

	// 2) Set MTU before applying config (1420 is optimal for WireGuard over typical networks)
	link, err := netlink.LinkByName(linkName)
	if err != nil {
		return logger.Errorf("failed to get link %s: %v", linkName, err)
	}
	if err := netlink.LinkSetMTU(link, 1420); err != nil {
		return logger.Errorf("failed to set MTU on %s: %v", linkName, err)
	}

	// 3) Apply WireGuard config
	if err := utils.SetWgConf(linkName, confPath); err != nil {
		return err
	}

	// 4) Create netns
	nsHandle, err := utils.CreateNetNS(netnsName)
	if err != nil {
		return err
	}
	defer nsHandle.Close()

	// 5) Per-netns DNS
	dnsServers, err := utils.ParseDNSFromConfig(confPath)
	if err != nil {
		dnsServers = []string{"1.1.1.1", "1.0.0.1"}
	}
	if err := utils.EnsureNetnsResolvConf(netnsName, dnsServers); err != nil {
		return err
	}

	// 6) Move iface into netns
	if err := utils.MoveIntToNS(nsHandle, linkName); err != nil {
		return err
	}

	// 7) Configure IP + routes inside netns
	h, err := netlink.NewHandleAt(nsHandle)
	if err != nil {
		return logger.Errorf("open handle: %v", err)
	}
	defer h.Close()

	if lo, err := h.LinkByName("lo"); err == nil {
		_ = h.LinkSetUp(lo)
	}

	ip, err := utils.ParseConfigForIP(confPath)
	if err != nil {
		return err
	}

	link, err = h.LinkByName(linkName)
	if err != nil {
		return logger.Errorf("find %q: %v", linkName, err)
	}

	addr, err := netlink.ParseAddr(ip)
	if err != nil {
		return logger.Errorf("parse addr %q: %v", ip, err)
	}
	if err := h.AddrReplace(link, addr); err != nil {
		return logger.Errorf("addr replace: %v", err)
	}
	if err := h.LinkSetUp(link); err != nil {
		return logger.Errorf("link up: %v", err)
	}

	// IPv4 default route
	v4Route := &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Dst: &net.IPNet{
			IP:   net.IPv4zero,
			Mask: net.CIDRMask(0, 32),
		},
	}
	if err := h.RouteReplace(v4Route); err != nil {
		return logger.Errorf("add default v4 route via %s: %v", linkName, err)
	}

	// IPv6 default (best-effort)
	v6Route := &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Dst: &net.IPNet{
			IP:   net.ParseIP("::"),
			Mask: net.CIDRMask(0, 128),
		},
	}
	_ = h.RouteReplace(v6Route)

	success = true
	logger.Info("WireGuard ready in netns " + logger.Highlight(netnsName))
	logger.Info("Interface " + logger.Highlight(linkName) + " address: " + logger.Highlight(ip))
	return nil
}

func init() {
	createCmd.Flags().StringVarP(&vpsName, "name", "n", "", "Custom name for the VPS instance (defaults to instance label)")

	rootCmd.AddCommand(createCmd)
}
