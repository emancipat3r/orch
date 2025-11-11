package cmd

import (
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/charmbracelet/keygen"
	"github.com/emancipat3r/vps3/logger"
	"github.com/emancipat3r/vps3/providers"
	"github.com/emancipat3r/vps3/ui"
	"github.com/emancipat3r/vps3/utils"
	"github.com/spf13/cobra"
	"github.com/vishvananda/netlink"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Provision a new VPS instance",
	Run: func(cmd *cobra.Command, args []string) {
		// Set up signal handling for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigChan
			logger.Info("\nOperation cancelled by user (Ctrl+C)")
			os.Exit(0)
		}()

		// Use wizard interface for provider selection and navigation
		provider := ui.ChoiceProvider()
		if provider == "" {
			logger.Info("Operation cancelled by user.")
			return
		}

		logger.Info("You selected: " + logger.Highlight(provider))

		// Load data for the selected provider and create navigable wizard
		switch provider {
		case "Linode":
			providerKey := providers.GetLinodeAPIKey(configFile, provider)
			accountBalance, err := providers.GetLinodesBalance(providerKey)

			if err != nil {
				logger.Error("Failed to get Linode account balance: " + err.Error())
				return
			}

			logger.Info("Linode account balance: " + logger.Highlight("$"+accountBalance))

			regions, err := providers.GetLinodeRegions()

			if err != nil {
				logger.Error("Failed to hit endpoint: " + err.Error())
				return
			}

			var regionOptions []string

			// Load all options for Linode
			for _, region := range regions {
				if region.Status == "ok" {
					regionOptions = append(regionOptions, region.ID+" - "+region.Label)
				}
			}

			images, err := providers.GetLinodeImages()
			if err != nil {
				logger.Error("Failed to get Linode images: " + err.Error())
				return
			}
			var imageOptions []string
			for _, image := range images {
				imageOptions = append(imageOptions, image.ID+" - "+image.Label)
			}

			resources, err := providers.GetLinodeResources()
			if err != nil {
				logger.Error("Failed to get Linode resources: " + err.Error())
				return
			}
			var resourceOptions []string
			for _, resource := range resources {
				resourceOptions = append(resourceOptions, resource.ID+" - "+resource.Label)
			}

			// Create navigable wizard with all options loaded
			var region, image, size string
			steps := []ui.WizardStep{
				{
					Key:         "region",
					Title:       "Select your region",
					Description: "",
					Options:     regionOptions,
					Value:       &region,
				},
				{
					Key:         "image",
					Title:       "Select your image",
					Description: "",
					Options:     imageOptions,
					Value:       &image,
				},
				{
					Key:         "size",
					Title:       "Select your resourcing",
					Description: "",
					Options:     resourceOptions,
					Value:       &size,
				},
			}

			selections, err := ui.CreateNavigableWizard(steps)
			if err != nil {
				logger.Info("Operation cancelled by user.")
				return
			}

			selectedRegion := selections["region"]
			selectedImage := selections["image"]
			selectedResource := selections["size"]

			logger.Info("You selected region: " + logger.Highlight(selectedRegion))
			logger.Info("You selected image: " + logger.Highlight(selectedImage))
			logger.Info("You selected type: " + logger.Highlight(selectedResource))

			selectedRegionSplit := strings.Split(selectedRegion, " ")
			selectedImageSplit := strings.Split(selectedImage, " ")
			selectedResourceSplit := strings.Split(selectedResource, " ")

			rootPassword, err := utils.GenerateRandomPassword(30)

			if err != nil {
				logger.Error("Failed to generate root password for Linode: " + err.Error())
				return
			}

			// === Per-instance keypair (using charmbracelet/keygen) ===
			keyName := "linode-" + providers.CreateUID()
			perPriv := pathSSH + keyName // ~/.config/vps/.ssh/linode-XXXXXXXX
			perPub := perPriv + ".pub"

			// optional: passphrase (store per key)
			pass, err := utils.GenerateRandomPassword(24)
			if err != nil {
				logger.Error("Failed to gen passphrase: " + err.Error())
				return
			}
			if err := os.WriteFile(pathSecrets+keyName+".pass", []byte(pass+"\n"), 0600); err != nil {
				logger.Error("Failed to store passphrase: " + err.Error())
				return
			}

			// create the keypair
			if _, err := keygen.New(perPriv, keygen.WithKeyType(keygen.Ed25519), keygen.WithPassphrase(pass), keygen.WithWrite()); err != nil {
				logger.Error("Failed to create per-instance keypair: " + err.Error())
				return
			}

			// Add key to ssh-agent if available
			if err := utils.AddKeyToSSHAgent(perPriv); err != nil {
				logger.Debug("Could not add key to ssh-agent: " + err.Error())
			}

			// Upload pubkey to Linode → get unique key ID
			keyID, err := providers.UploadLinodeSSHKey(providerKey, perPub)
			if err != nil {
				logger.Error("Failed to upload SSH key to Linode: " + err.Error())
				return
			}
			logger.Info("Per-instance SSH key ID: " + logger.Highlight(strconv.Itoa(keyID)))

			// Generate default vpsName if not provided
			if vpsName == "" {
				vpsName = "linode-" + providers.CreateUID()
			}

			// Create the instance with this key ID and persist priv key path
			logger.Info("Creating Linode...")
			_, err = providers.CreateLinode(
				providerKey,
				keyID,
				perPriv,
				selectedImageSplit[0],
				selectedRegionSplit[0],
				selectedResourceSplit[0],
				rootPassword,
				instanceFile,
				vpsName,
			)

			if err != nil {
				logger.Error("Failed to create Linode: " + err.Error())
			}

			if err := setupWireGuard(vpsName, vpsName); err != nil {
				logger.Error("WireGuard setup failed: " + err.Error())
			} else {
				logger.Info("WireGuard network namespace setup complete.")
			}

		case "DigitalOcean":
			providerKey := providers.GetDOAPIKey(configFile, provider)
			accountBalance, err := providers.GetDOBalance(providerKey)
			if err != nil {
				logger.Error("Failed to get DigitalOcean account balance: " + err.Error())
				return
			}
			logger.Info("DigitalOcean account balance: " + logger.Highlight("$"+accountBalance))

			// region
			regions, err := providers.GetDORegions(providerKey)
			if err != nil {
				logger.Error("Failed to hit endpoint: " + err.Error())
				return
			}
			var regionOptions []string
			for _, r := range regions {
				regionOptions = append(regionOptions, r.Slug+" - "+r.Name)
			}
			// Load all options for DigitalOcean
			images, err := providers.GetDOImages(providerKey)
			if err != nil {
				logger.Error("Failed to get DigitalOcean images: " + err.Error())
				return
			}
			var imageOptions []string
			for _, img := range images {
				imageOptions = append(imageOptions, img.Slug+" - "+img.Name)
			}

			sizes, err := providers.GetDOResources(providerKey)
			if err != nil {
				logger.Error("Failed to get DigitalOcean sizes: " + err.Error())
				return
			}
			var resourceOptions []string
			for _, s := range sizes {
				resourceOptions = append(resourceOptions, s.Slug+" - "+s.Name)
			}

			// Create navigable wizard with all options loaded
			var region, image, size string
			steps := []ui.WizardStep{
				{
					Key:         "region",
					Title:       "Select your region",
					Description: "",
					Options:     regionOptions,
					Value:       &region,
				},
				{
					Key:         "image",
					Title:       "Select your image",
					Description: "",
					Options:     imageOptions,
					Value:       &image,
				},
				{
					Key:         "size",
					Title:       "Select your resourcing",
					Description: "",
					Options:     resourceOptions,
					Value:       &size,
				},
			}

			selections, err := ui.CreateNavigableWizard(steps)
			if err != nil {
				logger.Info("Operation cancelled by user.")
				return
			}

			selectedRegion := selections["region"]
			selectedImage := selections["image"]
			selectedResource := selections["size"]

			logger.Info("You selected region: " + logger.Highlight(selectedRegion))
			logger.Info("You selected image: " + logger.Highlight(selectedImage))
			logger.Info("You selected type: " + logger.Highlight(selectedResource))

			selectedRegionSlug := strings.Split(selectedRegion, " ")[0]
			selectedImageSlug := strings.Split(selectedImage, " ")[0]
			selectedSizeSlug := strings.Split(selectedResource, " ")[0]

			// === Per-droplet keypair (using charmbracelet/keygen) ===
			keyName := "do-" + providers.CreateUID()
			perPriv := pathSSH + keyName // ~/.config/vps/.ssh/do-XXXXXXXX
			perPub := perPriv + ".pub"

			// optional: passphrase (store per key)
			pass, err := utils.GenerateRandomPassword(24)
			if err != nil {
				logger.Error("Failed to gen passphrase: " + err.Error())
				return
			}
			if err := os.WriteFile(pathSecrets+keyName+".pass", []byte(pass+"\n"), 0600); err != nil {
				logger.Error("Failed to store passphrase: " + err.Error())
				return
			}

			// create the keypair
			if _, err := keygen.New(perPriv, keygen.WithKeyType(keygen.Ed25519), keygen.WithPassphrase(pass), keygen.WithWrite()); err != nil {
				logger.Error("Failed to create per-droplet keypair: " + err.Error())
				return
			}

			// Add key to ssh-agent if available
			if err := utils.AddKeyToSSHAgent(perPriv); err != nil {
				logger.Debug("Could not add key to ssh-agent: " + err.Error())
			}

			// Upload pubkey to DigitalOcean → get unique key ID
			keyID, err := providers.UploadDOSSHKey(providerKey, perPub)
			if err != nil {
				logger.Error("Failed to upload SSH key to DigitalOcean: " + err.Error())
				return
			}
			logger.Info("Droplet SSH key ID: " + logger.Highlight(perPriv))

			// Generate default vpsName if not provided
			if vpsName == "" {
				vpsName = "digitalocean-" + providers.CreateUID()
			}

			// Create the droplet with this key ID and persist priv key path
			_, err = providers.CreateDroplet(
				providerKey,
				keyID,
				perPriv,
				selectedImageSlug,
				selectedRegionSlug,
				selectedSizeSlug,
				instanceFile,
				vpsName,
			)

			if err != nil {
				logger.Error("Failed to create Droplet: " + err.Error())
			}

			if err := setupWireGuard(vpsName, vpsName); err != nil {
				logger.Error("WireGuard setup failed: " + err.Error())
			} else {
				logger.Info("WireGuard network namespace setup complete.")
			}

		case "Vultr":
			providerKey := providers.GetVultrAPIKey(configFile, provider)
			accountBalance, err := providers.GetVultrBalance(providerKey)
			if err != nil {
				logger.Error("Failed to get Vultr account balance: " + err.Error())
				return
			}
			logger.Info("Vultr account balance: " + logger.Highlight("$"+accountBalance))

			// region
			regions, err := providers.GetVultrRegions(providerKey)
			if err != nil {
				logger.Error("Failed to hit endpoint: " + err.Error())
				return
			}
			var regionOptions []string
			for _, r := range regions {
				regionOptions = append(regionOptions, r.ID+" - "+r.City+", "+r.Country)
			}
			// Load all options for Vultr
			osImages, err := providers.GetVultrOS(providerKey)
			if err != nil {
				logger.Error("Failed to get Vultr OS images: " + err.Error())
				return
			}
			var osOptions []string
			for _, os := range osImages {
				osOptions = append(osOptions, strconv.Itoa(os.ID)+" - "+os.Name)
			}

			plans, err := providers.GetVultrPlans(providerKey)
			if err != nil {
				logger.Error("Failed to get Vultr plans: " + err.Error())
				return
			}
			var planOptions []string
			for _, p := range plans {
				planOptions = append(planOptions, p.ID+" - "+strconv.Itoa(p.VCPUCount)+" vCPU, "+strconv.Itoa(p.RAM)+" MB RAM, "+strconv.Itoa(p.Disk)+" GB SSD - $"+strconv.FormatFloat(p.MonthlyCost, 'f', 2, 64)+"/month")
			}

			// Create navigable wizard with all options loaded
			var region, os, plan string
			steps := []ui.WizardStep{
				{
					Key:         "region",
					Title:       "Select your region",
					Description: "",
					Options:     regionOptions,
					Value:       &region,
				},
				{
					Key:         "os",
					Title:       "Select your OS",
					Description: "",
					Options:     osOptions,
					Value:       &os,
				},
				{
					Key:         "plan",
					Title:       "Select your plan",
					Description: "",
					Options:     planOptions,
					Value:       &plan,
				},
			}

			selections, err := ui.CreateNavigableWizard(steps)
			if err != nil {
				logger.Info("Operation cancelled by user.")
				return
			}

			selectedRegion := selections["region"]
			selectedOS := selections["os"]
			selectedPlan := selections["plan"]

			logger.Info("You selected region: " + logger.Highlight(selectedRegion))
			logger.Info("You selected OS: " + logger.Highlight(selectedOS))
			logger.Info("You selected plan: " + logger.Highlight(selectedPlan))

			selectedRegionID := strings.Split(selectedRegion, " ")[0]
			selectedOSIDStr := strings.Split(selectedOS, " ")[0]
			selectedOSID, _ := strconv.Atoi(selectedOSIDStr)
			selectedPlanID := strings.Split(selectedPlan, " ")[0]

			// === Per-instance keypair (using charmbracelet/keygen) ===
			keyName := "vultr-" + providers.CreateUID()
			perPriv := pathSSH + keyName // ~/.config/vps/.ssh/vultr-XXXXXXXX
			perPub := perPriv + ".pub"

			// optional: passphrase (store per key)
			pass, err := utils.GenerateRandomPassword(24)
			if err != nil {
				logger.Error("Failed to gen passphrase: " + err.Error())
				return
			}
			if err := ioutil.WriteFile(pathSecrets+keyName+".pass", []byte(pass+"\n"), 0600); err != nil {
				logger.Error("Failed to store passphrase: " + err.Error())
				return
			}

			// create the keypair
			if _, err := keygen.New(perPriv, keygen.WithKeyType(keygen.Ed25519), keygen.WithPassphrase(pass), keygen.WithWrite()); err != nil {
				logger.Error("Failed to create per-instance keypair: " + err.Error())
				return
			}

			// Add key to ssh-agent if available
			if err := utils.AddKeyToSSHAgent(perPriv); err != nil {
				logger.Debug("Could not add key to ssh-agent: " + err.Error())
			}

			// Upload pubkey to Vultr → get unique key ID
			keyID, err := providers.UploadVultrSSHKey(providerKey, perPub)
			if err != nil {
				logger.Error("Failed to upload SSH key to Vultr: " + err.Error())
				return
			}
			logger.Info("Per-instance SSH key ID: " + logger.Highlight(keyID))

			// Generate default vpsName if not provided
			if vpsName == "" {
				vpsName = "vultr-" + providers.CreateUID()
			}

			logger.Info("Creating Vultr instance...")
			instanceID, err := providers.CreateVultrInstance(
				providerKey,
				keyID,
				perPriv,
				selectedOSID,
				selectedRegionID,
				selectedPlanID,
				instanceFile,
				vpsName,
			)

			if err != nil {
				logger.Error("Failed to create Vultr instance: " + err.Error())
			} else {
				logger.Info("Successfully created Vultr instance: " + logger.Highlight(instanceID))
			}

			if err := setupWireGuard(vpsName, vpsName); err != nil {
				logger.Error("WireGuard setup failed: " + err.Error())
			} else {
				logger.Info("WireGuard network namespace setup complete.")
			}

		default:
			logger.Warn("No provider was selected. Exiting...")
		}
	},
}

func setupWireGuard(vpsName, netnsName string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return logger.Errorf("failed to get home directory: %v", err)
	}

	confPath := filepath.Join(homeDir, ".config/vps/wg/"+vpsName+".conf")
	if _, err := os.Stat(confPath); err != nil {
		return logger.Errorf("WireGuard config not found at %s", confPath)
	}

	// If not specified, fall back to vpsName
	if netnsName == "" {
		netnsName = vpsName
	}

	logger.Info("Setting up WireGuard: " + logger.Highlight(netnsName))

	// 0) Idempotent: create-or-reuse WG link in root ns
 linkName := "wg-" + netnsName
	if _, err := netlink.LinkByName(linkName); err != nil {
		if _, cerr := utils.CreateWgInt(linkName); cerr != nil {
			return cerr
		}
	}

	// 1) Configure WG (keys/peers/port) in root ns
	if err := utils.SetWgConf(netnsName, confPath); err != nil {
		return err
	}

	// 2) Create (or reuse) namespace
	nsHandle, err := utils.CreateNetNS(netnsName)
	if err != nil {
		// If it already exists, try to open a handle to it
		// but netns lib doesn't have OpenNamed; CreateNamed is idempotent,
		// so just proceed on EEXIST-like situations.
		// If you want stricter behavior, add ExistsNetNS() in utils and branch.
		// For now, just surface the error.
		// (Most kernels let NewNamed be effectively idempotent via bind mount.)
		// If you want to be extra safe, ignore error if the mount already exists:
		// return nil on os.ErrExist.
		// Keeping it simple:
		return err
	}
	defer nsHandle.Close()

	// 3) Move interface into the namespace
	if err := utils.MoveIntToNS(nsHandle, vpsName); err != nil {
		return err
	}

	// 4) Assign IP (first Address from config) inside the namespace and bring link up
	ipCIDR, err := utils.ParseConfigForIP(confPath)
	if err != nil {
		return err
	}

	h, err := netlink.NewHandleAt(nsHandle)
	if err != nil {
		return logger.Errorf("open handle: %v", err)
	}
	defer h.Close()

	// Ensure loopback is up
	if lo, err := h.LinkByName("lo"); err == nil {
		_ = h.LinkSetUp(lo)
	}

	link, err := h.LinkByName(netnsName)
	if err != nil {
		return logger.Errorf("find %q: %v", netnsName, err)
	}

	addr, err := netlink.ParseAddr(ipCIDR)
	if err != nil {
		return logger.Errorf("parse addr %q: %v", ipCIDR, err)
	}
	if err := h.AddrReplace(link, addr); err != nil {
		return logger.Errorf("addr replace: %v", err)
	}
	if err := h.LinkSetUp(link); err != nil {
		return logger.Errorf("link up: %v", err)
	}

	logger.Info("WireGuard ready in netns " + logger.Highlight(netnsName))
	logger.Info("Interface address: " + logger.Highlight(ipCIDR))

	// Optional: add routes here if you want traffic steered through WG.
	// wg-quick normally installs routes; wgctrl does not.
	// Example default v4 (uncomment if desired):
	// _ = h.RouteReplace(&netlink.Route{
	//     LinkIndex: link.Attrs().Index,
	//     Dst: &net.IPNet{IP: net.IPv4zero, Mask: net.CIDRMask(0, 32)},
	// })

	return nil
}

func init() {
	createCmd.Flags().StringVarP(&vpsName, "name", "n", "", "Custom name for the VPS instance (defaults to instance label)")

	rootCmd.AddCommand(createCmd)
}

/*
 21:01:00 [INFO] ======================================================================
 21:01:00 [INFO] Ansible playbook complete
 21:01:01 [INFO] Setting up WireGuard: vps1
 21:01:01 [INFO] WireGuard interface vps1 created successfully
 21:01:01 [INFO] Applied WireGuard config to vps1
 21:01:01 [INFO] Created network namespace: vps1 (/var/run/netns/vps1)
 21:01:01 [ERROR] lookup "vps1" in source ns: Link not found
 21:01:01 [ERROR] WireGuard setup failed: lookup "vps1" in source ns: Link not found

  - Looks up vps1 interface in source ns
  8: vps1: <POINTOPOINT,NOARP> mtu 1420 qdisc noop state DOWN group default qlen 1000
      link/none

*/
