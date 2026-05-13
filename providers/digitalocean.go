package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
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

func GetDOAPIKey(configFile string, provider string) string {

	providerKey := utils.ParseCreds(configFile, provider)
	if providerKey == "" {
		logger.Error("Failed to parse provider credentials")
		return ""
	}
	return providerKey
}

type DOAccount struct {
	Balance string `json:"month_to_date_balance"`
}

func GetDOBalance(providerKey string) (string, error) {
	if providerKey == "" {
		return "", logger.Errorf("Provider key is empty")
	}

	req, err := http.NewRequest("GET", "https://api.digitalocean.com/v2/customers/my/balance", nil)
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

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Errors []struct {
				Reason string `json:"reason"`
			} `json:"errors"`
		}

		if err := json.Unmarshal(bodyBytes, &errResp); err == nil && len(errResp.Errors) > 0 {
			reason := errResp.Errors[0].Reason
			if reason == "Invalid Token" {
				return "", logger.Errorf("Your DigitalOcean key is invalid or expired. Check it in the config.")
			}

			return "", logger.Errorf("DigitalOcean API error: %s", reason)
		}
	}

	var account DOAccount
	err = json.Unmarshal(bodyBytes, &account)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s", account.Balance), nil

}

type DORegion struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type DORegionsResponse struct {
	Data []DORegion `json:"regions"`
}

func GetDORegions(providerKey string) ([]DORegion, error) {
	if providerKey == "" {
		return nil, logger.Errorf("Provider key is empty")
	}

	req, err := http.NewRequest("GET", "https://api.digitalocean.com/v2/regions", nil)
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

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Print(bodyBytes)
		return nil, logger.Errorf("Unexpected status code: %d | %s", resp.StatusCode, string(bodyBytes))
	}

	var regionsResp DORegionsResponse
	err = json.NewDecoder(resp.Body).Decode(&regionsResp)
	if err != nil {
		return nil, err
	}

	return regionsResp.Data, nil
}

type DOImage struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type DOImagesResponse struct {
	Data []DOImage `json:"images"`
}

func GetDOImages(providerKey string) ([]DOImage, error) {
	req, err := http.NewRequest("GET", "https://api.digitalocean.com/v2/images?page=1&per_page=0&type=distribution", nil)
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

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Print(bodyBytes)
		return nil, logger.Errorf("Unexpected status code: %d | %s", resp.StatusCode, string(bodyBytes))
	}

	var imagesResp DOImagesResponse
	err = json.NewDecoder(resp.Body).Decode(&imagesResp)
	if err != nil {
		return nil, err
	}

	return imagesResp.Data, nil
}

type DOResource struct {
	Name string `json:"description"`
	Slug string `json:"slug"`
}

type DOResourcesResponse struct {
	Data []DOResource `json:"sizes"`
}

func GetDOResources(providerKey string) ([]DOResource, error) {
	req, err := http.NewRequest("GET", "https://api.digitalocean.com/v2/sizes", nil)
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

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Print(bodyBytes)
		return nil, logger.Errorf("Unexpected status code: %d | %s", resp.StatusCode, string(bodyBytes))
	}

	var resourcesResp DOResourcesResponse
	err = json.NewDecoder(resp.Body).Decode(&resourcesResp)
	if err != nil {
		return nil, err
	}

	return resourcesResp.Data, nil
}

func CreateUID() string {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	num := rand.Intn(90000000) + 10000000
	numStr := strconv.Itoa(num)
	return numStr
}

func UploadDOSSHKey(providerKey, pubKeyPath string) (int, error) {
	if providerKey == "" {
		return 0, logger.Errorf("Provider key is empty")
	}

	data, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return 0, err
	}
	pubKey := strings.TrimSpace(string(data))

	keyName := CreateUID()

	payload := map[string]string{
		"name":       keyName,
		"public_key": pubKey,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequest("POST", "https://api.digitalocean.com/v2/account/keys", bytes.NewBuffer(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+providerKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusCreated {

		var parsed struct {
			SSHKey struct {
				ID int `json:"id"`
			} `json:"ssh_key"`
		}

		if err := json.Unmarshal(respBody, &parsed); err != nil {
			return 0, err
		}

		return parsed.SSHKey.ID, nil
	}

	if resp.StatusCode == http.StatusUnprocessableEntity {

		req2, _ := http.NewRequest("GET", "https://api.digitalocean.com/v2/account/keys", nil)
		req2.Header.Set("Authorization", "Bearer "+providerKey)
		req2.Header.Set("Accept", "application/json")

		resp2, err := client.Do(req2)

		if err != nil {
			return 0, err
		}
		defer resp2.Body.Close()

		body2, _ := io.ReadAll(resp2.Body)
		var parsed struct {
			Keys []struct {
				ID        int    `json:"id"`
				PublicKey string `json:"public_key"`
			} `json:"ssh_keys"`
		}

		if err := json.Unmarshal(body2, &parsed); err != nil {
			return 0, err
		}

		for _, k := range parsed.Keys {
			if strings.TrimSpace(k.PublicKey) == pubKey {
				return k.ID, nil
			}
		}
	}

	return 0, logger.Errorf("Failed to upload SSH key: status=%d body=%s", resp.StatusCode, string(respBody))
}

type DOCreateRequest struct {
	Name    string        `json:"name"`
	Region  string        `json:"region"`
	Size    string        `json:"size"`
	Image   string        `json:"image"`
	SSHKeys []interface{} `json:"ssh_keys,omitempty"` // allow int IDs
	Tags    []string      `json:"tags,omitempty"`
}

type DOCreateResponse struct {
	Droplet struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		CreatedAt string `json:"created_at"`
		SizeSlug  string `json:"size_slug"`
		Region    struct {
			Slug string `json:"slug"`
			Name string `json:"name"`
		} `json:"region"`
		Image struct {
			ID   int    `json:"id"`
			Slug string `json:"slug"`
		} `json:"image"`
		Networks struct {
			V4 []struct {
				IPAddress string `json:"ip_address"`
				Type      string `json:"type"`
			} `json:"v4"`
			V6 []struct {
				IPAddress string `json:"ip_address"`
				Type      string `json:"type"`
			} `json:"v6"`
		} `json:"networks"`
	} `json:"droplet"`
}

type DOInstance struct {
	Creation_Time string `toml:"created"`
	Id            string `toml:"id"`
	Host_Image    string `toml:"image"`
	Ipv4          string `toml:"ipv4"`
	Label         string `toml:"label"`
	Region        string `toml:"region"`
	Type          string `toml:"type"`
	KeyID         string `toml:"key_id"`
	PrivKeyPath   string `toml:"priv_key_path"`
	Provider      string `toml:"provider"`
	VPSName       string `toml:"vps_name"`
}

type DOInstancesToml map[string]DOInstance

func CreateDroplet(parent context.Context, providerKey string, sshKeyID int, privKeyPath, image, region, sizeSlug, instanceFile, vpsName string) (string, error) {
	if providerKey == "" {
		return "", logger.Errorf("Provider key is empty")
	}

	UID := CreateUID()
	payload := DOCreateRequest{
		Name:    UID,
		Region:  region,
		Size:    sizeSlug,
		Image:   image,
		SSHKeys: []interface{}{sshKeyID},
		Tags:    []string{"orch"},
	}

	b, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "https://api.digitalocean.com/v2/droplets", bytes.NewBuffer(b))
	req.Header.Set("Authorization", "Bearer "+providerKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	rb, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return "", logger.Errorf("Unexpected status code: %d | %s", resp.StatusCode, string(rb))
	}

	var parsed DOCreateResponse
	if err := json.Unmarshal(rb, &parsed); err != nil {
		return "", err
	}

	logger.Info("Droplet created with ID: " + logger.Highlight(strconv.Itoa(parsed.Droplet.ID)))

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
			// Check current droplet details every 10 seconds
			dropletReq, err := http.NewRequest("GET", "https://api.digitalocean.com/v2/droplets/"+strconv.Itoa(parsed.Droplet.ID), nil)
			if err != nil {
				time.Sleep(checkInterval)
				continue
			}
			dropletReq.Header.Set("Authorization", "Bearer "+providerKey)
			dropletReq.Header.Set("Accept", "application/json")

			dropletResp, err := http.DefaultClient.Do(dropletReq)
			if err != nil {
				time.Sleep(checkInterval)
				continue
			}

			dropletRb, err := io.ReadAll(dropletResp.Body)
			dropletResp.Body.Close()

			if err != nil {
				time.Sleep(checkInterval)
				continue
			}

			if dropletResp.StatusCode != http.StatusOK {
				time.Sleep(checkInterval)
				continue
			}

			var dropletData struct {
				Droplet struct {
					ID       int    `json:"id"`
					Status   string `json:"status"`
					Networks struct {
						V4 []struct {
							IPAddress string `json:"ip_address"`
							Type      string `json:"type"`
						} `json:"v4"`
					} `json:"networks"`
				} `json:"droplet"`
			}

			if err := json.Unmarshal(dropletRb, &dropletData); err != nil {
				time.Sleep(checkInterval)
				continue
			}

			// Check if IP is assigned and droplet is active
			for _, v4 := range dropletData.Droplet.Networks.V4 {
				if v4.Type == "public" && v4.IPAddress != "" && dropletData.Droplet.Status == "active" {
					finalIP = v4.IPAddress
					ui.FinishSpinner(spinnerProg, true, "")
					return
				}
			}

			// Update spinner message with current status
			ui.UpdateSpinnerMessage(spinnerProg, "Waiting for IP address assignment...")
			time.Sleep(checkInterval)
		}

		// Timeout reached
		ui.FinishSpinner(spinnerProg, false, "Timeout waiting for IP assignment")
	}()

	// Wait for the spinner to complete
	<-doneChan

	if finalIP == "" {
		// Try to get IP from initial response as fallback
		for _, v := range parsed.Droplet.Networks.V4 {
			if v.Type == "public" {
				finalIP = v.IPAddress
				break
			}
		}
		if finalIP == "" {
			finalIP = "pending"
		}
	}

	// Add small delay to ensure spinner is fully complete
	time.Sleep(100 * time.Millisecond)

	// Print VPS IP
	logger.Info("Instance IP: " + logger.Highlight(finalIP))

	vps := DOInstance{
		Creation_Time: parsed.Droplet.CreatedAt,
		Id:            strconv.Itoa(parsed.Droplet.ID),
		Host_Image:    parsed.Droplet.Image.Slug,
		Ipv4:          finalIP,
		Label:         parsed.Droplet.Name,
		Region:        parsed.Droplet.Region.Slug,
		Type:          parsed.Droplet.SizeSlug,
		KeyID:         strconv.Itoa(sshKeyID),
		PrivKeyPath:   privKeyPath,
		Provider:      "DigitalOcean",
		VPSName:       vpsName,
	}

	var all DOInstancesToml
	if _, err := os.Stat(instanceFile); err == nil {
		if _, err := toml.DecodeFile(instanceFile, &all); err != nil {
			return "", err
		}
	} else {
		all = make(DOInstancesToml)
	}

	all[vps.Id] = vps
	f, _ := os.Create(instanceFile)
	defer f.Close()
	if err := toml.NewEncoder(f).Encode(all); err != nil {
		return "", err
	}
	logger.Info("Updated VPS instance file with IP: " + logger.Highlight(instanceFile))

	// Run Ansible post-provisioning setup
	if vps.Ipv4 != "" && vps.Ipv4 != "pending" {
		if err := utils.SetupPostProvisioningAnsible(parent, vps.Ipv4, privKeyPath, vpsName); err != nil {
			logger.Warn("Post-provisioning setup failed: " + err.Error())
		}
	}

	return vps.Id, nil
}

func DeleteDOSSHKey(providerKey string, keyID int) error {
	req, _ := http.NewRequest("DELETE",
		fmt.Sprintf("https://api.digitalocean.com/v2/account/keys/%d", keyID), nil)
	req.Header.Set("Authorization", "Bearer "+providerKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(resp.Body)
		return logger.Errorf("SSH key deletion failed: %d %s", resp.StatusCode, string(b))
	}
	return nil
}

type DOInstances struct {
	Creation_Time string `json:"created_at"`
	Id            int    `json:"id"`
	Image         struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
	} `json:"image"`
	Networks struct {
		IPv4 []struct {
			IPAddress string `json:"ip_address"`
			Type      string `json:"type"`
		} `json:"v4"`
	} `json:"networks"`
	Region struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"region"`
	Size struct {
		Slug string `json:"slug"`
	} `json:"size"`
	Status string `json:"status"`
}

type DOInstancesResponse struct {
	Data []DOInstances `json:"droplets"`
}

func SelectDOInstance(providerKey string) ([]DOInstances, error) {
	if providerKey == "" {
		return nil, logger.Errorf("Provider key is empty")
	}

	req, err := http.NewRequest("GET", "https://api.digitalocean.com/v2/droplets?page=0&per_page=0", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+providerKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Print(bodyBytes)
		return nil, logger.Errorf("Unexpected status code: %d | %s", resp.StatusCode, string(bodyBytes))
	}

	var instancesResp DOInstancesResponse
	err = json.NewDecoder(resp.Body).Decode(&instancesResp)
	if err != nil {
		return nil, err
	}

	return instancesResp.Data, nil

}

func DestroyDroplet(providerKey, instanceID, instanceFile string) (string, error) {
	if providerKey == "" {
		return "", logger.Errorf("Provider key is empty")
	}

	// First, load the instance data to get the SSH key ID before deleting
	var instances DOInstancesToml
	if _, err := os.Stat(instanceFile); err == nil {
		if _, err := toml.DecodeFile(instanceFile, &instances); err != nil {
			return "", logger.Errorf("Failed to load instances file: %v", err)
		}
	} else {
		return "", logger.Errorf("Instance file not found: %s", instanceFile)
	}

	instance, exists := instances[instanceID]

	// Delete the droplet first
	req, err := http.NewRequest("DELETE", "https://api.digitalocean.com/v2/droplets/"+instanceID, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+providerKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	logger.Info("Deleted droplet: " + logger.Highlight(instanceID))

	// Delete the SSH key from DigitalOcean
	if instance.KeyID != "" && instance.KeyID != "0" {
		keyID, err := strconv.Atoi(instance.KeyID)
		if err != nil {
			logger.Warn("Invalid SSH key ID format: " + instance.KeyID)
		} else {
			err = DeleteDOSSHKey(providerKey, keyID)
			if err != nil {
				logger.Warn("Failed to delete SSH key from DigitalOcean: " + err.Error())
				// Don't return error here as droplet is already deleted
			} else {
				logger.Info("Deleted SSH key: " + logger.Highlight(instance.KeyID))
			}
		}
	}

	// Delete the private key file if it exists
	if instance.PrivKeyPath != "" {
		// Remove key from ssh-agent first
		if err := utils.RemoveKeyFromSSHAgent(instance.PrivKeyPath); err != nil {
			logger.Debug("Could not remove key from ssh-agent: " + err.Error())
		}

		if err := os.Remove(instance.PrivKeyPath); err != nil {
			logger.Warn("Failed to delete private key file: " + err.Error())
		} else {
			logger.Info("Deleted private key file: " + logger.Highlight(instance.PrivKeyPath))
		}

		// Delete the pub key file if it exists
		pubKeyFile := instance.PrivKeyPath + ".pub"
		if err := os.Remove(pubKeyFile); err != nil {
			// Don't warn about this as it might not exist
		} else {
			logger.Info("Deleted public key file: " + logger.Highlight(pubKeyFile))
		}

		// Also try to delete the passphrase file
		passPhraseFile := strings.Replace(instance.PrivKeyPath, "/.ssh/", "/secrets/", 1)
		passPhraseFile = strings.TrimSuffix(passPhraseFile, filepath.Ext(passPhraseFile)) + ".pass"
		if err := os.Remove(passPhraseFile); err != nil {
			// Don't warn about this as it might not exist
		} else {
			logger.Info("Deleted passphrase file: " + logger.Highlight(passPhraseFile))
		}
	}

	// Delete the WireGuard client config if it exists
	homeDir, err := os.UserHomeDir()
	if err == nil {
		var wgConfigPath string
		if instance.VPSName != "" {
			// Use VPSName if available (new instances)
			wgConfigPath = filepath.Join(homeDir, ".config", "vps", "wg", instance.VPSName+".conf")
		} else if instance.Ipv4 != "" && instance.Ipv4 != "pending" {
			// Fallback to IP-based name for backward compatibility
			wgConfigPath = filepath.Join(homeDir, ".config", "vps", "wg", "client-"+instance.Ipv4+".conf")
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

	// Remove the instance from the TOML file
	// _, exists := instances[instanceID]
	if exists {
		delete(instances, instanceID)

		// Write the updated instances file atomically
		tmp := instanceFile + ".tmp"
		f, err := os.Create(tmp)
		if err != nil {
			return "", logger.Errorf("Failed creating tmp instance file: %v", err)
		}

		if err := toml.NewEncoder(f).Encode(instances); err != nil {
			_ = f.Close()
			_ = os.Remove(tmp)
			return "", logger.Errorf("Failed updating instance file: %v", err)
		}

		if err := f.Close(); err != nil {
			_ = os.Remove(tmp)
			return "", logger.Errorf("Failed to close tmp instance file: %v", err)
		}

		if err := os.Rename(tmp, instanceFile); err != nil {
			_ = os.Remove(tmp)
			return "", logger.Errorf("Failed to rename tmp instance file: %v", err)
		}

		logger.Info("Updated instance file: " + logger.Highlight(instanceFile))

		return "", nil
	} else {
		logger.Warn("Instance not found in Instances file. Proceeding with destruction anyways...")
	}

	return "", nil

}

func ListDOInstancesTable(providerKey string) (string, error) {
	if providerKey == "" {
		return "", logger.Errorf("Provider key is empty")
	}

	req, err := http.NewRequest("GET", "https://api.digitalocean.com/v2/droplets?page=0&per_page=0", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+providerKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Print(bodyBytes)
		return "", logger.Errorf("Unexpected status code: %d | %s", resp.StatusCode, string(bodyBytes))
	}

	var instancesResp DOInstancesResponse
	err = json.NewDecoder(resp.Body).Decode(&instancesResp)
	if err != nil {
		return "", err
	}

	// Load existing instances from TOML file to check/update IP addresses
	instanceFile := os.Getenv("HOME") + "/.config/orch/instances/instances.toml"
	var storedInstances DOInstancesToml
	var hasChanges bool

	if _, err := os.Stat(instanceFile); err == nil {
		if _, err := toml.DecodeFile(instanceFile, &storedInstances); err != nil {
			logger.Warn("Failed to load instances file for IP sync: " + err.Error())
			storedInstances = make(DOInstancesToml)
		}
	} else {
		storedInstances = make(DOInstancesToml)
	}

	var rows [][]string
	for _, inst := range instancesResp.Data {
		var ipv4 string
		// Find the public IPv4 address
		for _, v4 := range inst.Networks.IPv4 {
			if v4.Type == "public" {
				ipv4 = v4.IPAddress
				break
			}
		}

		// Check if we need to update the stored instance with current IP
		instanceIDStr := strconv.Itoa(inst.Id)
		if storedInstance, exists := storedInstances[instanceIDStr]; exists {
			if storedInstance.Ipv4 != ipv4 && ipv4 != "" {
				// Update the stored instance with new IP
				storedInstance.Ipv4 = ipv4
				storedInstances[instanceIDStr] = storedInstance
				hasChanges = true
				logger.Info("Updated IP for droplet " + logger.Highlight(instanceIDStr) + ": " + logger.Highlight(ipv4))
			}
		}

		// Use image description if available, fallback to name, then slug
		imageName := inst.Image.Description
		if imageName == "" {
			imageName = inst.Image.Name
		}
		if imageName == "" {
			imageName = inst.Image.Slug
		}

		rows = append(rows, []string{
			strconv.Itoa(inst.Id),
			ipv4,
			inst.Region.Slug + " - " + inst.Region.Name,
			imageName,
			inst.Size.Slug,
			inst.Creation_Time,
			inst.Status,
		})
	}

	// Write back the instances file if we made changes
	if hasChanges {
		tmp := instanceFile + ".tmp"
		f, err := os.Create(tmp)
		if err != nil {
			logger.Warn("Failed to create temp file for IP sync: " + err.Error())
		} else {
			if err := toml.NewEncoder(f).Encode(storedInstances); err != nil {
				_ = f.Close()
				_ = os.Remove(tmp)
				logger.Warn("Failed to encode updated instances: " + err.Error())
			} else if err := f.Close(); err != nil {
				_ = os.Remove(tmp)
				logger.Warn("Failed to close temp instances file: " + err.Error())
			} else if err := os.Rename(tmp, instanceFile); err != nil {
				_ = os.Remove(tmp)
				logger.Warn("Failed to update instances file: " + err.Error())
			} else {
				logger.Info("Synced IP addresses to instances file")
			}
		}
	}

	fmt.Println(ui.InstanceTable(rows))

	return "", nil
}
