package cmd

import (
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/emancipat3r/vps3/logger"
	"github.com/emancipat3r/vps3/providers"
	"github.com/emancipat3r/vps3/ui"
	"github.com/emancipat3r/vps3/utils"
	"github.com/spf13/cobra"
	"github.com/vishvananda/netns"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy VPS instance(s)",
	Run: func(cmd *cobra.Command, args []string) {
		// Ctrl+C handling
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigChan
			logger.Info("\nOperation cancelled by user (Ctrl+C)")
			os.Exit(0)
		}()

		provider := ui.ChoiceProvider()
		if provider == "" {
			logger.Info("Operation cancelled by user.")
			return
		}

		u, err := user.Current()
		if err != nil {
			logger.Error("Failed to get current user: " + err.Error())
			return
		}

		pathConfig = filepath.Join(u.HomeDir, ".config/vps/config/")
		configFile = filepath.Join(pathConfig, "configuration.toml")
		pathInstances = filepath.Join(u.HomeDir, ".config/vps/instances/")
		instanceFile = filepath.Join(pathInstances, "instances.toml")

		switch provider {
		// ---------------- LINODE ----------------
		case "Linode":
			providerKey := providers.GetLinodeAPIKey(configFile, provider)

			accountBalance, err := providers.GetLinodesBalance(providerKey)
			if err != nil {
				logger.Error("Failed to get Linode account balance: " + err.Error())
				return
			}
			logger.Info("Linode account balance: " + logger.Highlight("$"+accountBalance))

			instances, err := providers.SelectLinodeInstance(providerKey)
			if err != nil {
				logger.Error("Failed to hit endpoint: " + err.Error())
				return
			}

			var instanceOptions []string
			for _, inst := range instances {
				instanceOptions = append(
					instanceOptions,
					inst.Creation_Time+" - "+strconv.Itoa(inst.Id)+" - "+inst.Host_Image+" - "+inst.Ipv4[0]+" - "+inst.Region,
				)
			}

			selectedInstances := ui.MultiSelect("Select the instance(s) to destroy:", instanceOptions)
			if len(selectedInstances) == 0 {
				logger.Info("No instances selected. Exiting...")
				return
			}

			logger.Info("You selected " + strconv.Itoa(len(selectedInstances)) + " instance(s) for destruction")
			for _, inst := range selectedInstances {
				logger.Info("  - " + logger.Highlight(inst))
			}

			if !ui.Confirm("Are you sure you want to proceed with destroying " + strconv.Itoa(len(selectedInstances)) + " instance(s)?") {
				logger.Info("Destruction cancelled by user.")
				return
			}

			for i, selected := range selectedInstances {
				parts := strings.Split(selected, " ")
				instanceID := parts[2]

				// Resolve vps_name BEFORE destroying
				vpsName, err := utils.GetVPSNameForInstance(instanceFile, instanceID)
				if err != nil {
					logger.Debug("No local vps_name mapping for Linode instance " + instanceID + ": " + err.Error())
				}

				logger.Info("(" + strconv.Itoa(i+1) + "/" + strconv.Itoa(len(selectedInstances)) +
					") Destroying Linode: " + logger.Highlight(instanceID))

				_, err = providers.DestroyLinode(providerKey, instanceID, instanceFile)
				if err != nil {
					logger.Error("Failed to destroy Linode " + instanceID + ": " + err.Error())
					continue
				}

				logger.Info("Successfully destroyed Linode: " + logger.Highlight(instanceID))

				if vpsName != "" {
					tearDownLocalByName(vpsName)
				}
			}

			logger.Info("Completed destruction of " + logger.Highlight(strconv.Itoa(len(selectedInstances))) + " Linode instance(s)")

		// ---------------- DIGITALOCEAN ----------------
		case "DigitalOcean":
			providerKey := providers.GetDOAPIKey(configFile, provider)
			accountBalance, err := providers.GetDOBalance(providerKey)
			if err != nil {
				logger.Error("Failed to get DigitalOcean account balance: " + err.Error())
				return
			}
			logger.Info("DigitalOcean account balance: " + logger.Highlight("$"+accountBalance))

			instances, err := providers.SelectDOInstance(providerKey)
			if err != nil {
				logger.Error("Failed to hit endpoint: " + err.Error())
				return
			}

			var instanceOptions []string
			for _, inst := range instances {
				var ipv4 string
				for _, v4 := range inst.Networks.IPv4 {
					if v4.Type == "public" {
						ipv4 = v4.IPAddress
						break
					}
				}

				imageName := inst.Image.Description
				if imageName == "" {
					imageName = inst.Image.Name
				}
				if imageName == "" {
					imageName = inst.Image.Slug
				}

				instanceOptions = append(
					instanceOptions,
					inst.Creation_Time+" - "+strconv.Itoa(inst.Id)+" - "+imageName+" - "+ipv4+" - "+inst.Region.Slug,
				)
			}

			selectedInstances := ui.MultiSelect("Select the instance(s) to destroy:", instanceOptions)
			if len(selectedInstances) == 0 {
				logger.Info("No instances selected. Exiting...")
				return
			}

			logger.Info("You selected " + strconv.Itoa(len(selectedInstances)) + " instance(s) for destruction")
			for _, inst := range selectedInstances {
				logger.Info("  - " + logger.Highlight(inst))
			}

			if !ui.Confirm("Are you sure you want to proceed with destroying " + strconv.Itoa(len(selectedInstances)) + " instance(s)?") {
				logger.Info("Destruction cancelled by user.")
				return
			}

			for i, selected := range selectedInstances {
				parts := strings.Split(selected, " ")
				instanceID := parts[2]

				// Resolve vps_name BEFORE destroying
				vpsName, err := utils.GetVPSNameForInstance(instanceFile, instanceID)
				if err != nil {
					logger.Debug("No local vps_name mapping for DO instance " + instanceID + ": " + err.Error())
				}

				logger.Info("(" + strconv.Itoa(i+1) + "/" + strconv.Itoa(len(selectedInstances)) +
					") Destroying droplet: " + logger.Highlight(instanceID))

				_, err = providers.DestroyDroplet(providerKey, instanceID, instanceFile)
				if err != nil {
					logger.Error("Failed to destroy droplet " + instanceID + ": " + err.Error())
					continue
				}

				logger.Info("Successfully destroyed droplet: " + logger.Highlight(instanceID))

				if vpsName != "" {
					tearDownLocalByName(vpsName)
				}
			}

			logger.Info("Completed destruction of " + logger.Highlight(strconv.Itoa(len(selectedInstances))) + " DigitalOcean instance(s)")

		// ---------------- VULTR ----------------
		case "Vultr":
			providerKey := providers.GetVultrAPIKey(configFile, provider)
			accountBalance, err := providers.GetVultrBalance(providerKey)
			if err != nil {
				logger.Error("Failed to get Vultr account balance: " + err.Error())
				return
			}
			logger.Info("Vultr account balance: " + logger.Highlight("$"+accountBalance))

			instances, err := providers.SelectVultrInstance(providerKey)
			if err != nil {
				logger.Error("Failed to hit endpoint: " + err.Error())
				return
			}

			regions, err := providers.GetVultrRegions(providerKey)
			regionCache := make(map[string]string)
			if err == nil {
				for _, r := range regions {
					regionCache[r.ID] = r.ID + " - " + r.City + ", " + r.Country
				}
			}

			var instanceOptions []string
			for _, inst := range instances {
				regionDisplay := inst.Region
				if verboseRegion, ok := regionCache[inst.Region]; ok {
					regionDisplay = verboseRegion
				}
				instanceOptions = append(
					instanceOptions,
					inst.DateCreated+" - "+inst.ID+" - "+inst.OS+" - "+inst.MainIP+" - "+regionDisplay,
				)
			}

			selectedInstances := ui.MultiSelect("Select the instance(s) to destroy:", instanceOptions)
			if len(selectedInstances) == 0 {
				logger.Info("No instances selected. Exiting...")
				return
			}

			logger.Info("You selected " + strconv.Itoa(len(selectedInstances)) + " instance(s) for destruction")
			for _, inst := range selectedInstances {
				logger.Info("  - " + logger.Highlight(inst))
			}

			if !ui.Confirm("Are you sure you want to proceed with destroying " + strconv.Itoa(len(selectedInstances)) + " instance(s)?") {
				logger.Info("Destruction cancelled by user.")
				return
			}

			for i, selected := range selectedInstances {
				parts := strings.Split(selected, " ")
				instanceID := parts[2]

				// Resolve vps_name BEFORE destroying
				vpsName, err := utils.GetVPSNameForInstance(instanceFile, instanceID)
				if err != nil {
					logger.Debug("No local vps_name mapping for Vultr instance " + instanceID + ": " + err.Error())
				}

				logger.Info("(" + strconv.Itoa(i+1) + "/" + strconv.Itoa(len(selectedInstances)) +
					") Destroying Vultr instance: " + logger.Highlight(instanceID))

				err = providers.DestroyVultr(providerKey, instanceID, instanceFile)
				if err != nil {
					logger.Error("Failed to destroy Vultr instance " + instanceID + ": " + err.Error())
					continue
				}

				logger.Info("Successfully destroyed Vultr instance: " + logger.Highlight(instanceID))

				if vpsName != "" {
					tearDownLocalByName(vpsName)
				}
			}

			logger.Info("Completed destruction of " + logger.Highlight(strconv.Itoa(len(selectedInstances))) + " Vultr instance(s)")

		default:
			logger.Warn("No provider was selected. Exiting...")
		}
	},
}

func tearDownLocalByName(nsName string) {
	logger.Info("Tearing down local WireGuard/netns for " + logger.Highlight(nsName))

	if err := utils.TearDownNamespace(nsName, netns.None()); err != nil {
		logger.Error("Failed to tear down netns " + nsName + ": " + err.Error())
	}

	nsDir := filepath.Join("/etc/netns", nsName)
	_ = os.Remove(filepath.Join(nsDir, "resolv.conf"))
	_ = os.Remove(nsDir)

	u, err := user.Current()
	if err == nil {
		wgConf := filepath.Join(u.HomeDir, ".config/vps/wg", nsName+".conf")
		_ = os.Remove(wgConf)
	}

	logger.Info("Local WireGuard/netns cleaned up for " + logger.Highlight(nsName))
}

func init() {
	rootCmd.AddCommand(destroyCmd)
}
