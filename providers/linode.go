package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/emancipat3r/orch/logger"
	"github.com/emancipat3r/orch/ui"
	"github.com/emancipat3r/orch/utils"
)

type LinodeAccount struct {
	Balance           float64 `json:"balance"`
	BalanceUninvoiced float64 `json:"balance_uninvoiced"`
}

func GetLinodesBalance(providerKey string) (string, error) {
	if providerKey == "" {
		return "", logger.Errorf("Provider key is empty")
	}

	req, err := http.NewRequest("GET", "https://api.linode.com/v4/account", nil)
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
				return "", logger.Errorf("Your Linode key is invalid or expired. Check it in the config.")
			}

			return "", logger.Errorf("Linode API error: %s", reason)
		}
	}

	var account LinodeAccount
	err = json.Unmarshal(bodyBytes, &account)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%.2f", account.Balance), nil

}

type LinodeRegion struct {
	ID      string `json:"id"`
	Country string `json:"country"`
	Label   string `json:"label"`
	Status  string `json:"status"`
}

type RegionsResponse struct {
	Data []LinodeRegion `json:"data"`
}

func GetLinodeRegions() ([]LinodeRegion, error) {
	req, err := http.NewRequest("GET", "https://api.linode.com/v4/regions", nil)
	if err != nil {
		return nil, err
	}
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

	var regionsResp RegionsResponse
	err = json.NewDecoder(resp.Body).Decode(&regionsResp)
	if err != nil {
		return nil, err
	}

	return regionsResp.Data, nil
}

type LinodeImage struct {
	Label string `json:"label"`
	ID    string `json:"id"`
}

type LinodeImagesResponse struct {
	Data []LinodeImage `json:"data"`
}

func GetLinodeImages() ([]LinodeImage, error) {
	req, err := http.NewRequest("GET", "https://api.linode.com/v4/images?page=1&page_size=100", nil)
	if err != nil {
		return nil, err
	}
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

	var imagesResp LinodeImagesResponse
	err = json.NewDecoder(resp.Body).Decode(&imagesResp)
	if err != nil {
		return nil, err
	}

	return imagesResp.Data, nil
}

type LinodeResource struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type LinodeResourcesResponse struct {
	Data []LinodeResource `json:"data"`
}

func GetLinodeResources() ([]LinodeResource, error) {
	req, err := http.NewRequest("GET", "https://api.linode.com/v4/linode/types", nil)
	if err != nil {
		return nil, err
	}
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

	var resourcesResp LinodeResourcesResponse
	err = json.NewDecoder(resp.Body).Decode(&resourcesResp)
	if err != nil {
		return nil, err
	}

	return resourcesResp.Data, nil
}

type LinodeCreateRequest struct {
	Booted         bool     `json:"booted"`
	SwapSize       int      `json:"swap_size"`
	AuthorizedKeys []string `json:"authorized_keys"`
	Image          string   `json:"image"`
	Region         string   `json:"region"`
	Type           string   `json:"type"`
	RootPass       string   `json:"root_pass"`
}

type linodeResponseJSONBytes struct {
	Creation_Time string   `json:"created"`
	Host_UUID     string   `json:"host_uuid"`
	Id            int      `json:"id"`
	Host_Image    string   `json:"image"`
	Ipv4          []string `json:"ipv4"`
	Ipv6          string   `json:"ipv6"`
	Label         string   `json:"label"`
	Region        string   `json:"region"`
	Type          string   `json:"type"`
}

type LinodeInstance struct {
	Creation_Time string `toml:"created"`
	Host_UUID     string `toml:"host_uuid"`
	Id            string `toml:"id"`
	Host_Image    string `toml:"image"`
	Ipv4          string `toml:"ipv4"`
	Ipv6          string `toml:"ipv6"`
	Label         string `toml:"label"`
	Region        string `toml:"region"`
	Type          string `toml:"type"`
	SSHKeyID      string `toml:"ssh_key_id"`
	PrivKeyPath   string `toml:"priv_key_path"`
	Provider      string `toml:"provider"`
	VPSName       string `toml:"vps_name"`
}

// LinodeSSHKeyRequest for uploading SSH keys
type LinodeSSHKeyRequest struct {
	Label  string `json:"label"`
	SSHKey string `json:"ssh_key"`
}

// LinodeSSHKeyResponse from SSH key creation
type LinodeSSHKeyResponse struct {
	ID     int    `json:"id"`
	Label  string `json:"label"`
	SSHKey string `json:"ssh_key"`
}

// UploadLinodeSSHKey uploads an SSH key to Linode and returns the key ID
func UploadLinodeSSHKey(providerKey, pubKeyPath string) (int, error) {
	if providerKey == "" {
		return 0, logger.Errorf("Provider key is empty")
	}

	data, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return 0, err
	}
	pubKey := strings.TrimSpace(string(data))

	keyName := "orch-" + CreateUID()
	payload := LinodeSSHKeyRequest{
		Label:  keyName,
		SSHKey: pubKey,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return 0, logger.Errorf("Failed to marshal SSH key request: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.linode.com/v4/profile/sshkeys", bytes.NewBuffer(jsonBody))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+providerKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode == http.StatusOK {
		var parsed LinodeSSHKeyResponse
		if err := json.Unmarshal(respBody, &parsed); err != nil {
			return 0, logger.Errorf("Failed to parse SSH key response: %v", err)
		}
		return parsed.ID, nil
	}

	return 0, logger.Errorf("Failed to upload SSH key: status=%d body=%s", resp.StatusCode, string(respBody))
}

// DeleteLinodeSSHKey deletes an SSH key from Linode
func DeleteLinodeSSHKey(providerKey string, keyID int) error {
	if providerKey == "" {
		return logger.Errorf("Provider key is empty")
	}

	req, err := http.NewRequest("DELETE", fmt.Sprintf("https://api.linode.com/v4/profile/sshkeys/%d", keyID), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+providerKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return logger.Errorf("SSH key deletion failed: %d %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func CreateLinode(parent context.Context, providerKey string, sshKeyID int, privKeyPath, image, region, resource, rootPass, instanceFile, vpsName string) (string, error) {
	if providerKey == "" {
		return "", logger.Errorf("Provider key is empty")
	}

	// Read the public key content from the private key path
	pubKeyPath := privKeyPath + ".pub"
	data, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return "", logger.Errorf("Failed to read public key file: %v", err)
	}
	pubKeyContent := strings.TrimSpace(string(data))

	payload := LinodeCreateRequest{
		Booted:         true,
		SwapSize:       512,
		AuthorizedKeys: []string{pubKeyContent},
		Image:          image,
		Region:         region,
		Type:           resource,
		RootPass:       rootPass,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		logger.Errorf("Failed to marshal request payload: %v", err)
	}

	request, err := http.NewRequest("POST", "https://api.linode.com/v4/linode/instances", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+providerKey)
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	responseBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	var parsedResponseBytes linodeResponseJSONBytes
	err = json.Unmarshal(responseBytes, &parsedResponseBytes)
	if err != nil {
		return "", err
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		return "", logger.Errorf("Unexpected status code: %s | %s", strconv.Itoa(response.StatusCode), string(responseBytes))
	}

	logger.Info("Instance created with ID: " + logger.Highlight(strconv.Itoa(parsedResponseBytes.Id)))

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
			instanceReq, err := http.NewRequest("GET", "https://api.linode.com/v4/linode/instances/"+strconv.Itoa(parsedResponseBytes.Id), nil)
			if err != nil {
				time.Sleep(checkInterval)
				continue
			}
			instanceReq.Header.Set("Authorization", "Bearer "+providerKey)
			instanceReq.Header.Set("Accept", "application/json")

			instanceResp, err := client.Do(instanceReq)
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

			var instanceData linodeResponseJSONBytes
			if err := json.Unmarshal(instanceRb, &instanceData); err != nil {
				time.Sleep(checkInterval)
				continue
			}

			// Check if IP is assigned and we have a public IP
			if len(instanceData.Ipv4) > 0 && instanceData.Ipv4[0] != "" {
				finalIP = instanceData.Ipv4[0]
				ui.FinishSpinner(spinnerProg, true, "")
				return
			}

			// Update spinner message
			ui.UpdateSpinnerMessage(spinnerProg, "Still waiting for IP assignment...")
			time.Sleep(checkInterval)
		}

		// Timeout reached
		ui.FinishSpinner(spinnerProg, false, "Timeout waiting for IP assignment")
	}()

	// Wait for the spinner to complete
	<-doneChan

	if finalIP == "" {
		if len(parsedResponseBytes.Ipv4) > 0 {
			finalIP = parsedResponseBytes.Ipv4[0]
		} else {
			finalIP = "pending"
		}
	}

	// Add small delay to ensure spinner is fully cleared
	time.Sleep(100 * time.Millisecond)

	// Output the IP using logger
	logger.Info("Instance IP: " + logger.Highlight(finalIP))

	VPS := LinodeInstance{
		Creation_Time: parsedResponseBytes.Creation_Time,
		Host_UUID:     parsedResponseBytes.Host_UUID,
		Id:            strconv.Itoa(parsedResponseBytes.Id),
		Host_Image:    parsedResponseBytes.Host_Image,
		Ipv4:          finalIP,
		Ipv6:          parsedResponseBytes.Ipv6,
		Label:         parsedResponseBytes.Label,
		Region:        parsedResponseBytes.Region,
		Type:          parsedResponseBytes.Type,
		SSHKeyID:      strconv.Itoa(sshKeyID),
		PrivKeyPath:   privKeyPath,
		Provider:      "Linode",
		VPSName:       vpsName,
	}

	if err := utils.UpsertInstanceRecord(instanceFile, VPS.Id, VPS); err != nil {
		return "", logger.Errorf("update instance file: %v", err)
	}

	logger.Info("Updated VPS instance file with IP: " + logger.Highlight(instanceFile))

	// Run Ansible post-provisioning setup
	if finalIP != "" && finalIP != "pending" {
		if err := utils.SetupPostProvisioningAnsible(parent, finalIP, privKeyPath, vpsName); err != nil {
			logger.Warn("Post-provisioning setup failed: " + err.Error())
		}
	}

	return strconv.Itoa(parsedResponseBytes.Id), nil
}

type LinodeInstances struct {
	Creation_Time string   `json:"created"`
	Id            int      `json:"id"`
	Host_Image    string   `json:"image"`
	Ipv4          []string `json:"ipv4"`
	Ipv6          string   `json:"ipv6"`
	Region        string   `json:"region"`
	Type          string   `json:"type"`
	Status        string   `json:"status"`
}

type LinodeInstancesResponse struct {
	Data []LinodeInstances `json:"data"`
}

func SelectLinodeInstance(providerKey string) ([]LinodeInstances, error) {
	if providerKey == "" {
		return nil, logger.Errorf("Provider key is empty")
	}

	req, err := http.NewRequest("GET", "https://api.linode.com/v4/linode/instances?page_size=500", nil)
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

	var instancesResp LinodeInstancesResponse
	err = json.NewDecoder(resp.Body).Decode(&instancesResp)
	if err != nil {
		return nil, err
	}

	return instancesResp.Data, nil

}

// DestroyLinode deletes the Linode and its uploaded SSH key. A 404 on the
// instance is treated as already destroyed so local cleanup can still proceed.
func DestroyLinode(providerKey, instanceID, sshKeyID string) error {
	if providerKey == "" {
		return logger.Errorf("Provider key is empty")
	}

	req, err := http.NewRequest("DELETE", "https://api.linode.com/v4/linode/instances/"+instanceID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+providerKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		logger.Info("Deleted Linode instance: " + logger.Highlight(instanceID))
	case http.StatusNotFound:
		logger.Warn("Linode instance " + instanceID + " no longer exists remotely; cleaning up locally")
	default:
		rb, _ := io.ReadAll(resp.Body)
		return logger.Errorf("Unexpected status code: %d | %s", resp.StatusCode, string(rb))
	}

	deleteRemoteSSHKey("Linode", sshKeyID, func(id int) error { return DeleteLinodeSSHKey(providerKey, id) })
	return nil
}
