package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/emancipat3r/orch/logger"
	"github.com/emancipat3r/orch/ui"
	"github.com/emancipat3r/orch/utils"
)

type VultrAccount struct {
	Balance           float64 `json:"balance"`
	PendingCharges    float64 `json:"pending_charges"`
	LastPaymentDate   string  `json:"last_payment_date"`
	LastPaymentAmount float64 `json:"last_payment_amount"`
}

type VultrAccountResponse struct {
	Account VultrAccount `json:"account"`
}

func GetVultrBalance(providerKey string) (string, error) {
	if providerKey == "" {
		return "", logger.Errorf("Provider key is empty")
	}

	req, err := http.NewRequest("GET", "https://api.vultr.com/v2/account", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+providerKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	rb, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", logger.Errorf("Unexpected status code: %d | %s", resp.StatusCode, string(rb))
	}

	var parsed VultrAccountResponse
	if err := json.Unmarshal(rb, &parsed); err != nil {
		return "", err
	}

	return strconv.FormatFloat(parsed.Account.Balance, 'f', 2, 64), nil
}

type VultrRegion struct {
	ID        string   `json:"id"`
	City      string   `json:"city"`
	Country   string   `json:"country"`
	Continent string   `json:"continent"`
	Options   []string `json:"options"`
}

type VultrRegionsResponse struct {
	Regions []VultrRegion `json:"regions"`
}

func GetVultrRegions(providerKey string) ([]VultrRegion, error) {
	if providerKey == "" {
		return nil, logger.Errorf("Provider key is empty")
	}

	req, err := http.NewRequest("GET", "https://api.vultr.com/v2/regions", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+providerKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rb, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, logger.Errorf("Unexpected status code: %d | %s", resp.StatusCode, string(rb))
	}

	var parsed VultrRegionsResponse
	if err := json.Unmarshal(rb, &parsed); err != nil {
		return nil, err
	}

	return parsed.Regions, nil
}

type VultrOS struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Arch   string `json:"arch"`
	Family string `json:"family"`
}

type VultrOSResponse struct {
	OS []VultrOS `json:"os"`
}

func GetVultrOS(providerKey string) ([]VultrOS, error) {
	if providerKey == "" {
		return nil, logger.Errorf("Provider key is empty")
	}

	req, err := http.NewRequest("GET", "https://api.vultr.com/v2/os", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+providerKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rb, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, logger.Errorf("Unexpected status code: %d | %s", resp.StatusCode, string(rb))
	}

	var parsed VultrOSResponse
	if err := json.Unmarshal(rb, &parsed); err != nil {
		return nil, err
	}

	return parsed.OS, nil
}

type VultrPlan struct {
	ID          string   `json:"id"`
	VCPUCount   int      `json:"vcpu_count"`
	RAM         int      `json:"ram"`
	Disk        int      `json:"disk"`
	Bandwidth   int      `json:"bandwidth"`
	MonthlyCost float64  `json:"monthly_cost"`
	Type        string   `json:"type"`
	Locations   []string `json:"locations"`
}

type VultrPlansResponse struct {
	Plans []VultrPlan `json:"plans"`
}

func GetVultrPlans(providerKey string) ([]VultrPlan, error) {
	if providerKey == "" {
		return nil, logger.Errorf("Provider key is empty")
	}

	req, err := http.NewRequest("GET", "https://api.vultr.com/v2/plans", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+providerKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rb, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, logger.Errorf("Unexpected status code: %d | %s", resp.StatusCode, string(rb))
	}

	var parsed VultrPlansResponse
	if err := json.Unmarshal(rb, &parsed); err != nil {
		return nil, err
	}

	return parsed.Plans, nil
}

type VultrSSHKey struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	SSHKey      string `json:"ssh_key"`
	DateCreated string `json:"date_created"`
}

type VultrSSHKeysResponse struct {
	SSHKeys []VultrSSHKey `json:"ssh_keys"`
}

type VultrSSHKeyCreateRequest struct {
	Name   string `json:"name"`
	SSHKey string `json:"ssh_key"`
}

type VultrSSHKeyCreateResponse struct {
	SSHKey VultrSSHKey `json:"ssh_key"`
}

func UploadVultrSSHKey(providerKey string, pubKeyPath string) (string, error) {
	if providerKey == "" {
		return "", logger.Errorf("Provider key is empty")
	}

	data, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return "", err
	}

	keyName := "orch-" + CreateUID()
	payload := VultrSSHKeyCreateRequest{
		Name:   keyName,
		SSHKey: string(data),
	}

	b, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", "https://api.vultr.com/v2/ssh-keys", bytes.NewBuffer(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+providerKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	rb, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		return "", logger.Errorf("Unexpected status code: %d | %s", resp.StatusCode, string(rb))
	}

	var parsed VultrSSHKeyCreateResponse
	if err := json.Unmarshal(rb, &parsed); err != nil {
		return "", err
	}

	return parsed.SSHKey.ID, nil
}

type VultrCreateRequest struct {
	Region    string   `json:"region"`
	Plan      string   `json:"plan"`
	OSID      int      `json:"os_id"`
	Label     string   `json:"label"`
	SSHKeyIDs []string `json:"sshkey_id"`
	Tags      []string `json:"tags"`
	Hostname  string   `json:"hostname"`
}

type VultrInstance struct {
	ID               string   `json:"id"`
	OS               string   `json:"os"`
	RAM              int      `json:"ram"`
	Disk             int      `json:"disk"`
	MainIP           string   `json:"main_ip"`
	VCPUCount        int      `json:"vcpu_count"`
	Region           string   `json:"region"`
	Plan             string   `json:"plan"`
	DateCreated      string   `json:"date_created"`
	Status           string   `json:"status"`
	AllowedBandwidth int      `json:"allowed_bandwidth"`
	NetmaskV4        string   `json:"netmask_v4"`
	GatewayV4        string   `json:"gateway_v4"`
	PowerStatus      string   `json:"power_status"`
	ServerStatus     string   `json:"server_status"`
	V6Network        string   `json:"v6_network"`
	V6MainIP         string   `json:"v6_main_ip"`
	V6NetworkSize    int      `json:"v6_network_size"`
	Label            string   `json:"label"`
	InternalIP       string   `json:"internal_ip"`
	KVM              string   `json:"kvm"`
	Hostname         string   `json:"hostname"`
	OSID             int      `json:"os_id"`
	AppID            int      `json:"app_id"`
	ImageID          string   `json:"image_id"`
	FirewallGroupID  string   `json:"firewall_group_id"`
	Features         []string `json:"features"`
	Tags             []string `json:"tags"`
	DefaultPassword  string   `json:"default_password"`
	UserScheme       string   `json:"user_scheme"`
}

type VultrCreateResponse struct {
	Instance VultrInstance `json:"instance"`
}

type VultrInstanceRecord struct {
	Creation_Time   string `toml:"creation_time"`
	Id              string `toml:"id"`
	Host_Image      string `toml:"host_image"`
	Ipv4            string `toml:"ipv4"`
	Label           string `toml:"label"`
	Region          string `toml:"region"`
	Type            string `toml:"type"`
	SSHKeyID        string `toml:"ssh_key_id"`
	PrivKeyPath     string `toml:"priv_key_path"`
	DefaultPassword string `toml:"default_password"`
	Provider        string `toml:"provider"`
	VPSName         string `toml:"vps_name"`
}

type VultrInstancesToml map[string]VultrInstanceRecord

func CreateVultrInstance(parent context.Context, providerKey, sshKeyID, privKeyPath string, osID int, region, plan, instanceFile, vpsName string) (string, error) {
	if providerKey == "" {
		return "", logger.Errorf("Provider key is empty")
	}

	UID := CreateUID()
	payload := VultrCreateRequest{
		Region:    region,
		Plan:      plan,
		OSID:      osID,
		Label:     "orch-" + UID,
		SSHKeyIDs: []string{sshKeyID},
		Tags:      []string{"orch"},
		Hostname:  "orch-" + UID,
	}

	b, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", "https://api.vultr.com/v2/instances", bytes.NewBuffer(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+providerKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	rb, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusAccepted {
		return "", logger.Errorf("Unexpected status code: %d | %s", resp.StatusCode, string(rb))
	}

	var parsed VultrCreateResponse
	if err := json.Unmarshal(rb, &parsed); err != nil {
		return "", err
	}

	logger.Info("Instance created with ID: " + logger.Highlight(parsed.Instance.ID))

	// Start the spinner. Parent context so a user cancel aborts polling.
	ctx, cancel := context.WithTimeout(parent, 5*time.Minute)
	defer cancel()

	spinnerProg, doneChan := ui.IPWaitSpinner(ctx, "Waiting for IP address assignment...")

	// Wait for IP address to be assigned
	var finalIP string
	maxWaitTime := 5 * time.Minute
	checkInterval := 10 * time.Second
	startTime := time.Now()

	go func() {
		defer close(doneChan)

		for time.Since(startTime) < maxWaitTime {
			// Get current instance details
			instanceReq, err := http.NewRequest("GET", "https://api.vultr.com/v2/instances/"+parsed.Instance.ID, nil)
			if err != nil {
				time.Sleep(checkInterval)
				continue
			}
			instanceReq.Header.Set("Authorization", "Bearer "+providerKey)
			instanceReq.Header.Set("Accept", "application/json")

			instanceResp, err := http.DefaultClient.Do(instanceReq)
			if err != nil {
				time.Sleep(checkInterval)
				continue
			}

			instanceRb, err := io.ReadAll(instanceResp.Body)
			instanceResp.Body.Close()

			if err != nil {
				time.Sleep(checkInterval)
				continue
			}

			if instanceResp.StatusCode != http.StatusOK {
				time.Sleep(checkInterval)
				continue
			}

			var instanceData struct {
				Instance VultrInstance `json:"instance"`
			}

			if err := json.Unmarshal(instanceRb, &instanceData); err != nil {
				time.Sleep(checkInterval)
				continue
			}

			// Check if IP is assigned and instance is active
			if instanceData.Instance.MainIP != "" && instanceData.Instance.Status == "active" {
				finalIP = instanceData.Instance.MainIP
				ui.FinishSpinner(spinnerProg, true, "")
				return
			}

			// Update spinner message with current status
			ui.UpdateSpinnerMessage(spinnerProg, "Status: Waiting for IP address assignment..")
			time.Sleep(checkInterval)
		}

		// Timeout reached
		ui.FinishSpinner(spinnerProg, false, "Timeout waiting for IP assignment")
	}()

	// Wait for the spinner to complete
	<-doneChan

	if finalIP == "" {
		finalIP = "pending"
	}

	// Add small delay to ensure spinner is fully cleared
	time.Sleep(100 * time.Millisecond)

	// Output the IP using logger
	logger.Info("Instance IP: " + logger.Highlight(finalIP))

	vps := VultrInstanceRecord{
		Creation_Time:   parsed.Instance.DateCreated,
		Id:              parsed.Instance.ID,
		Host_Image:      strconv.Itoa(osID),
		Ipv4:            finalIP,
		Label:           parsed.Instance.Label,
		Region:          parsed.Instance.Region,
		Type:            parsed.Instance.Plan,
		SSHKeyID:        sshKeyID,
		PrivKeyPath:     privKeyPath,
		DefaultPassword: parsed.Instance.DefaultPassword,
		Provider:        "Vultr",
		VPSName:         vpsName,
	}

	var all VultrInstancesToml
	if _, err := os.Stat(instanceFile); err == nil {
		if _, err := toml.DecodeFile(instanceFile, &all); err != nil {
			logger.Warn("Could not decode existing instances file, starting fresh: " + err.Error())
			all = make(VultrInstancesToml)
		}
	} else {
		all = make(VultrInstancesToml)
	}

	all[parsed.Instance.ID] = vps
	f, _ := os.Create(instanceFile)
	defer f.Close()
	if err := toml.NewEncoder(f).Encode(all); err != nil {
		return "", err
	}
	logger.Info("Updated VPS instance file with IP: " + logger.Highlight(instanceFile))

	// Run Ansible post-provisioning setup
	if finalIP != "" && finalIP != "pending" {
		if err := utils.SetupPostProvisioningAnsible(parent, finalIP, privKeyPath, vpsName); err != nil {
			logger.Warn("Post-provisioning setup failed: " + err.Error())
		}
	}

	return parsed.Instance.ID, nil
}

func DeleteVultrSSHKey(providerKey string, keyID string) error {
	if providerKey == "" {
		return logger.Errorf("Provider key is empty")
	}

	req, err := http.NewRequest("DELETE", "https://api.vultr.com/v2/ssh-keys/"+keyID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+providerKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		rb, _ := io.ReadAll(resp.Body)
		return logger.Errorf("Unexpected status code: %d | %s", resp.StatusCode, string(rb))
	}

	return nil
}

type VultrInstancesListResponse struct {
	Instances []VultrInstance `json:"instances"`
}

func ListVultrInstances(providerKey string) ([]VultrInstance, error) {
	if providerKey == "" {
		return nil, logger.Errorf("Provider key is empty")
	}

	req, err := http.NewRequest("GET", "https://api.vultr.com/v2/instances", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+providerKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rb, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, logger.Errorf("Unexpected status code: %d | %s", resp.StatusCode, string(rb))
	}

	var parsed VultrInstancesListResponse
	if err := json.Unmarshal(rb, &parsed); err != nil {
		return nil, err
	}

	return parsed.Instances, nil
}

func DestroyVultrInstance(providerKey string, instanceID string) error {
	if providerKey == "" {
		return logger.Errorf("Provider key is empty")
	}

	req, err := http.NewRequest("DELETE", "https://api.vultr.com/v2/instances/"+instanceID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+providerKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		rb, _ := io.ReadAll(resp.Body)
		return logger.Errorf("Unexpected status code: %d | %s", resp.StatusCode, string(rb))
	}

	return nil
}

func SelectVultrInstance(providerKey string) ([]VultrInstance, error) {
	return ListVultrInstances(providerKey)
}

func DestroyVultr(providerKey string, instanceID string, instanceFile string) error {
	if providerKey == "" {
		return logger.Errorf("Provider key is empty")
	}

	// Load Vultr instances to get SSH key ID before deleting
	var instances VultrInstancesToml
	if _, err := os.Stat(instanceFile); err == nil {
		if _, err := toml.DecodeFile(instanceFile, &instances); err != nil {
			return logger.Errorf("Failed to load instances file: %v", err)
		}
	} else {
		return logger.Errorf("No instances file found")
	}

	// Get SSH key ID and private key path for cleanup
	var sshKeyID string
	var privKeyPath string
	instance, exists := instances[instanceID]
	if exists {
		sshKeyID = instance.SSHKeyID
		privKeyPath = instance.PrivKeyPath
	}

	// Delete the instance
	err := DestroyVultrInstance(providerKey, instanceID)
	if err != nil {
		return logger.Errorf("Failed to destroy Vultr instance: %v", err)
	}

	// Clean up SSH key if we have the ID
	if sshKeyID != "" {
		if err := DeleteVultrSSHKey(providerKey, sshKeyID); err != nil {
			logger.Warn("Failed to delete SSH key: " + err.Error())
		} else {
			logger.Info("Cleaned up SSH key: " + logger.Highlight(sshKeyID))
		}
	}

	// Delete the private key file if it exists
	if privKeyPath != "" {
		// Remove key from ssh-agent first
		if err := utils.RemoveKeyFromSSHAgent(privKeyPath); err != nil {
			logger.Debug("Could not remove key from ssh-agent: " + err.Error())
		}

		if err := os.Remove(privKeyPath); err != nil {
			logger.Warn("Failed to delete private key file: " + err.Error())
		} else {
			logger.Info("Deleted private key file: " + logger.Highlight(privKeyPath))
		}

		// Delete the pub key file if it exists
		pubKeyFile := privKeyPath + ".pub"
		if err := os.Remove(pubKeyFile); err != nil {
			// Don't warn about this as it might not exist
		} else {
			logger.Info("Deleted public key file: " + logger.Highlight(pubKeyFile))
		}

		// Also try to delete the passphrase file
		passPhraseFile := strings.Replace(privKeyPath, "/.ssh/", "/secrets/", 1)
		passPhraseFile = strings.TrimSuffix(passPhraseFile, filepath.Ext(passPhraseFile)) + ".pass"
		if err := os.Remove(passPhraseFile); err != nil {
			// Don't warn about this as it might not exist
		} else {
			logger.Info("Deleted passphrase file: " + logger.Highlight(passPhraseFile))
		}
	}

	// Delete the WireGuard client config if it exists
	if exists {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			var wgConfigPath string
			if instance.VPSName != "" {
				// Use VPSName if available (new instances)
				wgConfigPath = filepath.Join(homeDir, ".config", "orch", "wg", instance.VPSName+".conf")
			} else if instance.Ipv4 != "" && instance.Ipv4 != "pending" {
				// Fallback to IP-based name for backward compatibility
				wgConfigPath = filepath.Join(homeDir, ".config", "orch", "wg", "client-"+instance.Ipv4+".conf")
			}

			if wgConfigPath != "" {
				if err := os.Remove(wgConfigPath); err != nil {
					// Check if file exists before warning
					if !os.IsNotExist(err) {
						logger.Warn("Failed to delete WireGuard config: " + err.Error())
					}
				} else {
					logger.Info("Deleted WireGuard config: " + logger.Highlight(wgConfigPath))
				}
			}
		}
	}

	// Remove from local instances file
	if len(instances) > 0 {
		delete(instances, instanceID)
		f, err := os.Create(instanceFile)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := toml.NewEncoder(f).Encode(instances); err != nil {
			return err
		}
		logger.Info("Updated VPS instance file: " + logger.Highlight(instanceFile))
	}

	logger.Info("Successfully destroyed Vultr instance: " + logger.Highlight(instanceID))
	return nil
}
