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
	IPv6    bool          `json:"ipv6,omitempty"`
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
	Id            int    `toml:"id"`
	Host_Image    string `toml:"image"`
	Ipv4          string `toml:"ipv4"`
	Ipv6          string `toml:"ipv6"`
	Label         string `toml:"label"`
	Region        string `toml:"region"`
	Type          string `toml:"type"`
	KeyID         int    `toml:"key_id"`
	PrivKeyPath   string `toml:"priv_key_path"`
}

type DOInstancesToml map[string]DOInstance

func CreateDroplet(providerKey string, sshKeyID int, privKeyPath, image, region, sizeSlug, instanceFile string) (string, error) {
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
		IPv6:    true,
		Tags:    []string{"vps3"},
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

	ipv4 := ""
	for _, v := range parsed.Droplet.Networks.V4 {
		if v.Type == "public" {
			ipv4 = v.IPAddress
			break
		}
	}
	ipv6 := ""
	for _, v := range parsed.Droplet.Networks.V6 {
		if v.Type == "public" {
			ipv6 = v.IPAddress
			break
		}
	}

	vps := DOInstance{
		Creation_Time: parsed.Droplet.CreatedAt,
		Id:            parsed.Droplet.ID,
		Host_Image:    parsed.Droplet.Image.Slug,
		Ipv4:          ipv4,
		Ipv6:          ipv6,
		Label:         parsed.Droplet.Name,
		Region:        parsed.Droplet.Region.Slug,
		Type:          parsed.Droplet.SizeSlug,
		KeyID:         sshKeyID,
		PrivKeyPath:   privKeyPath,
	}

	var all DOInstancesToml
	if _, err := os.Stat(instanceFile); err == nil {
		if _, err := toml.DecodeFile(instanceFile, &all); err != nil {
			return "", err
		}
	} else {
		all = make(DOInstancesToml)
	}

	idStr := strconv.Itoa(vps.Id)
	all[idStr] = vps
	f, _ := os.Create(instanceFile)
	defer f.Close()
	if err := toml.NewEncoder(f).Encode(all); err != nil {
		return "", err
	}
	logger.Info("Updated VPS instance file: " + logger.Highlight(instanceFile))
	return idStr, nil
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
		Slug string `json:"slug"`
	} `json:"image"`
	Networks struct {
		IPv4 []struct {
			IPAddress string `json:"ip_address"`
		} `json:"v4"`
	} `json:"networks"`
	Ipv6   string `json:"ipv6"`
	Region struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"region"`
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

	logger.Success("Deleted droplet: " + logger.Highlight(instanceID))

	// remove instanceID table in instance toml file
	err = DeleteByTableName(instanceFile, instanceID)

	if err != nil {
		return "", logger.Errorf("Failed to update the instanceFile: %s", instanceFile)
	}

	return "", nil

}
