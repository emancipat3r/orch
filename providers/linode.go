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

	"github.com/BurntSushi/toml"
	"github.com/emancipat3r/vps3/logger"
	"github.com/emancipat3r/vps3/ui"
	"github.com/emancipat3r/vps3/utils"
)

func GetLinodeAPIKey(configFile string, provider string) string {

	providerKey := utils.ParseCreds(configFile, provider)
	if providerKey == "" {
		logger.Error("Failed to parse provider credentials")
		return ""
	}
	return providerKey
}

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

func ListLinodes(providerKey string) (string, error) {
	if providerKey == "" {
		return "", logger.Errorf("Provider key is empty")
	}

	request, err := http.NewRequest("GET", "https://api.linode.com/v4/linode/instances?page=1&page_size=100", nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/json")

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

	if response.StatusCode != http.StatusOK {
		return "", logger.Errorf("Unexpected status code: %s | %s", strconv.Itoa(response.StatusCode), string(responseBytes))
	}

	formattedResponse := utils.PrettyPrintJSON(responseBytes)
	return formattedResponse, nil
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
	Provider      string `toml:"provider"`
}

type LinodeInstancesToml map[string]LinodeInstance

func CreateLinode(providerKey, pubKeyPath, image, region, resource, rootPass, instanceFile string) (string, error) {
	if providerKey == "" {
		return "", logger.Errorf("Provider key is empty")
	}

	data, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return "", err
	}

	dataString := string(data)
	dataStringStripped := strings.TrimSpace(dataString)

	payload := LinodeCreateRequest{
		Booted:         true,
		SwapSize:       512,
		AuthorizedKeys: []string{dataStringStripped},
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

	// Start the spinner
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
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
		Provider:      "Linode",
	}

	var allinstances LinodeInstancesToml

	if _, err := os.Stat(instanceFile); err == nil {
		if _, err := toml.DecodeFile(instanceFile, &allinstances); err != nil {
			logger.Error("Unable to load pre-existing instance file: " + logger.Highlight(instanceFile) + string(err.Error()))
			return "", err
		}
	} else {
		allinstances = make(LinodeInstancesToml)
	}

	allinstances[VPS.Id] = VPS

	f, err := os.Create(instanceFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	err = toml.NewEncoder(f).Encode(allinstances)
	if err != nil {
		return "", err
	}

	logger.Info("Updated VPS instance file with IP: " + logger.Highlight(instanceFile))

	return "", nil
}

type instanceJSONBytes struct {
	Creation_Time string   `json:"created"`
	Id            int      `json:"id"`
	Host_Image    string   `json:"image"`
	Ipv4          []string `json:"ipv4"`
	Ipv6          string   `json:"ipv6"`
	Region        string   `json:"region"`
	Type          string   `json:"type"`
	Status        string   `json:"status"`
}

type linodeListResponse struct {
	Data    []instanceJSONBytes `json:"data"`
	Page    int                 `json:"page"`
	Pages   int                 `json:"pages"`
	Results int                 `json:"results"`
}

func ListLinodeInstancesTable(providerKey string, instanceFile string) (string, error) {
	if providerKey == "" {
		return "", logger.Errorf("Provider key is empty")
	}

	req, err := http.NewRequest("GET", "https://api.linode.com/v4/linode/instances?page_size=500", nil)
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

	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var parsedResponseBytes linodeListResponse
	err = json.Unmarshal(responseBytes, &parsedResponseBytes)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Print(bodyBytes)
		return "", logger.Errorf("Unexpected status code: %d | %s", resp.StatusCode, string(bodyBytes))
	}

	// Cache regions for verbose display
	regionCache := make(map[string]string)
	regions, err := GetLinodeRegions()
	if err == nil {
		for _, region := range regions {
			regionCache[region.ID] = region.ID + " - " + region.Label
		}
	}

	// Load existing instances from TOML file to check/update IP addresses
	var storedInstances LinodeInstancesToml
	var hasChanges bool

	if _, err := os.Stat(instanceFile); err == nil {
		if _, err := toml.DecodeFile(instanceFile, &storedInstances); err != nil {
			logger.Warn("Failed to load instances file for IP sync: " + err.Error())
			storedInstances = make(LinodeInstancesToml)
		}
	} else {
		storedInstances = make(LinodeInstancesToml)
	}

	var rows [][]string
	for _, inst := range parsedResponseBytes.Data {
		ipv4 := ""
		if len(inst.Ipv4) > 0 {
			ipv4 = inst.Ipv4[0]
		}

		// Check if we need to update the stored instance with current IP
		instanceIDStr := strconv.Itoa(inst.Id)
		if storedInstance, exists := storedInstances[instanceIDStr]; exists {
			if storedInstance.Ipv4 != ipv4 && ipv4 != "" {
				// Update the stored instance with new IP
				storedInstance.Ipv4 = ipv4
				storedInstances[instanceIDStr] = storedInstance
				hasChanges = true
				logger.Info("Updated IP for Linode instance " + logger.Highlight(instanceIDStr) + ": " + logger.Highlight(ipv4))
			}
		}

		// Get verbose region name
		regionDisplay := inst.Region
		if verboseRegion, exists := regionCache[inst.Region]; exists {
			regionDisplay = verboseRegion
		}

		rows = append(rows, []string{
			strconv.Itoa(inst.Id),
			ipv4,
			regionDisplay,
			inst.Host_Image,
			inst.Type,
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
				logger.Success("Synced IP addresses to instances file")
			}
		}
	}

	fmt.Println(ui.InstanceTable(rows))

	return "", nil

}

type LinodeInstances struct {
	Creation_Time string   `json:"created"`
	Id            int      `json:"id"`
	Host_Image    string   `json:"image"`
	Ipv4          []string `json:"ipv4"`
	Ipv6          string   `json:"ipv6"`
	Region        string   `json:"region"`
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

// Delete by table name (your table header equals the instance ID string)
func DeleteByTableName(instanceFile string, instanceID ...string) error {
	// Load instanceFile
	var m LinodeInstancesToml

	if _, err := os.Stat(instanceFile); err != nil {
		return logger.Errorf("Cannot load instances file, not found: %s", instanceFile)
	}

	if _, err := toml.DecodeFile(instanceFile, &m); err != nil {
		return logger.Errorf("decode toml: %v", err)
	}

	if m == nil {
		return nil
	}

	// Delete tables associated with instanceIDs
	for _, ID := range instanceID {
		delete(m, ID)
	}

	// Write atomically (tmp + rename)
	tmp := instanceFile + ".tmp"
	f, err := os.Create(tmp)

	if err != nil {
		return logger.Errorf("Failed creating tmp instance file: %v", err)
	}

	if err := toml.NewEncoder(f).Encode(m); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return logger.Errorf("Failed updating instance file: %v", err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("Failed to close tmp instance file: %w", err)
	}

	return os.Rename(tmp, instanceFile)
}

func DestroyLinode(providerKey, instanceID string) (string, error) {
	if providerKey == "" {
		return "", logger.Errorf("Provider key is empty")
	}

	req, err := http.NewRequest("DELETE", "https://api.linode.com/v4/linode/instances/"+instanceID, nil)
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

	logger.Success("Deleted Linode instance: " + logger.Highlight(instanceID))

	return "", nil

}
