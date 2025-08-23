package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/emancipat3r/vps3/logger"
	"github.com/emancipat3r/vps3/utils"
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

/*
{
  "images": [
    {
      "id": 135125666,
      "name": "9 Stream x64",
      "distribution": "CentOS",
      "slug": "centos-stream-9-x64",
      "public": true,
      "regions": [
        "tor1",
        "syd1",
        "sgp1",
        "sfo3",
        "sfo2",
        "sfo1",
        "nyc3",
        "nyc2",
        "nyc1",
        "lon1",
        "fra1",
        "blr1",
        "atl1",
        "ams3",
        "ams2"
      ],
      "created_at": "2023-06-22T20:26:46Z",
      "min_disk_size": 10,
      "type": "base",
      "size_gigabytes": 0.5,
      "description": "CentOS Stream 9 x64",
      "tags": [],
      "status": "available",
      "error_message": ""
    },
*/

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

// 8 number UID
func CreateInstanceUID() string {
	rand.New(rand.NewSource(time.Now().UnixNano()))

	num := rand.Intn(90000000) + 10000000 // always 8 digits
	numStr := strconv.Itoa(num)
	return numStr
}

type DOCreateRequest struct {
	Name     string   `json:"name"`
	SSHKeys  []string `json:"ssh_keys"`
	Image    string   `json:"image"`
	Region   string   `json:"region"`
	Type     string   `json:"type"`
	RootPass string   `json:"root_pass"`
}

type DOresponseJSONBytes struct {
	Creation_Time string   `json:"created"`
	Host_UUID     string   `json:"host_uuid"`
	Name          int      `json:"name"`
	Host_Image    string   `json:"image"`
	Ipv4          []string `json:"ipv4"`
	Ipv6          string   `json:"ipv6"`
	Label         string   `json:"label"`
	Region        string   `json:"region"`
	Type          string   `json:"type"`
}

type DOInstance struct {
	Creation_Time string `toml:"created"`
	Host_UUID     string `toml:"host_uuid"`
	Id            int    `toml:"id"`
	Host_Image    string `toml:"image"`
	Ipv4          string `toml:"ipv4"`
	Ipv6          string `toml:"ipv6"`
	Label         string `toml:"label"`
	Region        string `toml:"region"`
	Type          string `toml:"type"`
}

type DOInstancesToml map[string]DOInstance

func CreateDroplet(providerKey, pubKeyPath, image, region, resource, instanceFile string) (string, error) {
	if providerKey == "" {
		return "", logger.Errorf("Provider key is empty")
	}

	data, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return "", err
	}

	dataString := string(data)
	dataStringStripped := strings.TrimSpace(dataString)

	UID := CreateInstanceUID()

	payload := DOCreateRequest{
		Name:    UID,
		Region:  region,
		Image:   image,
		Type:    resource,
		SSHKeys: []string{dataStringStripped},
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		logger.Errorf("Failed to marshal request payload: %v", err)
	}

	request, err := http.NewRequest("POST", "https://api.digitalocean.com/v2/droplets", bytes.NewBuffer(jsonBody))
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

	logger.Debug(string(responseBytes))

	var parsedResponseBytes DOresponseJSONBytes
	err = json.Unmarshal(responseBytes, &parsedResponseBytes)
	if err != nil {
		return "", err
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		return "", logger.Errorf("Unexpected status code: %s | %s", strconv.Itoa(response.StatusCode), string(responseBytes))
	}

	VPS := DOInstance{
		Creation_Time: parsedResponseBytes.Creation_Time,
		Host_UUID:     parsedResponseBytes.Host_UUID,
		Id:            parsedResponseBytes.Name,
		Host_Image:    parsedResponseBytes.Host_Image,
		Ipv4:          parsedResponseBytes.Ipv4[0],
		Ipv6:          parsedResponseBytes.Ipv6,
		Label:         parsedResponseBytes.Label,
		Region:        parsedResponseBytes.Region,
		Type:          parsedResponseBytes.Type,
	}

	var allinstances DOInstancesToml

	if _, err := os.Stat(instanceFile); err == nil {
		if _, err := toml.DecodeFile(instanceFile, &allinstances); err != nil {
			logger.Error("Unable to load pre-existing instance file: " + logger.Highlight(instanceFile) + string(err.Error()))
			return "", err
		}
	} else {
		allinstances = make(DOInstancesToml)
	}

	vpsIDStr := strconv.Itoa(VPS.Id)
	allinstances[vpsIDStr] = VPS

	f, err := os.Create(instanceFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	err = toml.NewEncoder(f).Encode(allinstances)
	if err != nil {
		return "", err
	}

	logger.Info("Updated VPS instance file: " + logger.Highlight(instanceFile))
	if err != nil {
		return "", err
	}

	return "", nil
}
