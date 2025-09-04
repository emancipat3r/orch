package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/emancipat3r/vps3/logger"
	"github.com/emancipat3r/vps3/ui"
	"github.com/emancipat3r/vps3/utils"
)

func GetVultrAPIKey(configFile string, provider string) string {
	providerKey := utils.ParseCreds(configFile, provider)
	if providerKey == "" {
		logger.Error("Failed to parse provider credentials")
		return ""
	}
	return providerKey
}

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

	keyName := "vps3-" + CreateUID()
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
}

type VultrInstancesToml map[string]VultrInstanceRecord

func CreateVultrInstance(providerKey string, sshKeyID string, privKeyPath string, osID int, region string, plan string, instanceFile string) (string, error) {
	// Use separate instance file for Vultr to avoid conflicts with other providers
	vultrInstanceFile := strings.Replace(instanceFile, "instances.toml", "vultr-instances.toml", 1)
	if providerKey == "" {
		return "", logger.Errorf("Provider key is empty")
	}

	UID := CreateUID()
	payload := VultrCreateRequest{
		Region:    region,
		Plan:      plan,
		OSID:      osID,
		Label:     "vps3-" + UID,
		SSHKeyIDs: []string{sshKeyID},
		Tags:      []string{"vps3"},
		Hostname:  "vps3-" + UID,
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

	vps := VultrInstanceRecord{
		Creation_Time:   parsed.Instance.DateCreated,
		Id:              parsed.Instance.ID,
		Host_Image:      parsed.Instance.OS,
		Ipv4:            parsed.Instance.MainIP,
		Label:           parsed.Instance.Label,
		Region:          parsed.Instance.Region,
		Type:            parsed.Instance.Plan,
		SSHKeyID:        sshKeyID,
		PrivKeyPath:     privKeyPath,
		DefaultPassword: parsed.Instance.DefaultPassword,
	}

	var all VultrInstancesToml
	if _, err := os.Stat(vultrInstanceFile); err == nil {
		if _, err := toml.DecodeFile(vultrInstanceFile, &all); err != nil {
			logger.Warn("Could not decode existing Vultr instances file, starting fresh: " + err.Error())
			all = make(VultrInstancesToml)
		}
	} else {
		all = make(VultrInstancesToml)
	}

	all[parsed.Instance.ID] = vps
	f, _ := os.Create(vultrInstanceFile)
	defer f.Close()
	if err := toml.NewEncoder(f).Encode(all); err != nil {
		return "", err
	}
	logger.Info("Updated VPS instance file: " + logger.Highlight(vultrInstanceFile))
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
	// Use separate instance file for Vultr
	vultrInstanceFile := strings.Replace(instanceFile, "instances.toml", "vultr-instances.toml", 1)

	if providerKey == "" {
		return logger.Errorf("Provider key is empty")
	}

	// Load Vultr instances to get SSH key ID before deleting
	var instances VultrInstancesToml
	if _, err := os.Stat(vultrInstanceFile); err == nil {
		if _, err := toml.DecodeFile(vultrInstanceFile, &instances); err != nil {
			logger.Warn("Could not load Vultr instances file: " + err.Error())
		}
	}

	// Get SSH key ID for cleanup
	var sshKeyID string
	if instance, exists := instances[instanceID]; exists {
		sshKeyID = instance.SSHKeyID
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

	// Remove from local instances file
	if len(instances) > 0 {
		delete(instances, instanceID)
		f, err := os.Create(vultrInstanceFile)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := toml.NewEncoder(f).Encode(instances); err != nil {
			return err
		}
		logger.Info("Updated VPS instance file: " + logger.Highlight(vultrInstanceFile))
	}

	logger.Info("Successfully destroyed Vultr instance: " + logger.Highlight(instanceID))
	return nil
}

func ListVultrInstancesTable(providerKey string) (string, error) {
	if providerKey == "" {
		return "", logger.Errorf("Provider key is empty")
	}

	instances, err := ListVultrInstances(providerKey)
	if err != nil {
		return "", err
	}

	if len(instances) == 0 {
		logger.Info("No instances found")
		return "", nil
	}

	var rows [][]string
	for _, instance := range instances {
		rows = append(rows, []string{
			instance.ID,
			instance.MainIP,
			instance.Region,
			instance.OS,
			instance.Plan,
			instance.DateCreated,
			instance.Status,
		})
	}

	fmt.Println(ui.InstanceTable(rows))

	return "", nil
}
