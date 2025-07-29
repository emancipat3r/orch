package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/emancipat3r/vps3/logger"

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

type Account struct {
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

			return "", logger.Errorf("Linode API error %s", reason)
		}
	}

	var account Account
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

type Region struct {
	ID      string `json:"id"`
	Country string `json:"country"`
	Label   string `json:"label"`
	Status  string `json:"status"`
}

type RegionsResponse struct {
	Data []Region `json:"data"`
}

func GetLinodeRegions() ([]Region, error) {
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

type Image struct {
	Label string `json:"label"`
	ID    string `json:"id"`
}

type ImagesResponse struct {
	Data []Image `json:"data"`
}

func GetLinodeImages() ([]Image, error) {
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

	var imagesResp ImagesResponse
	err = json.NewDecoder(resp.Body).Decode(&imagesResp)
	if err != nil {
		return nil, err
	}

	return imagesResp.Data, nil
}

type Resource struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type ResourcesResponse struct {
	Data []Resource `json:"data"`
}

func GetLinodeResources() ([]Resource, error) {
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

	var resourcesResp ResourcesResponse
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

type responseJSONBytes struct {
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

type Instance struct {
	Creation_Time string   `toml:"created"`
	Host_UUID     string   `toml:"host_uuid"`
	Id            int      `toml:"id"`
	Host_Image    string   `toml:"image"`
	Ipv4          []string `toml:"ipv4"`
	Ipv6          string   `toml:"ipv6"`
	Label         string   `toml:"label"`
	Region        string   `toml:"region"`
	Type          string   `toml:"type"`
}

type InstancesToml map[string]Instance

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
	fmt.Println(dataStringStripped)

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

	var parsedResponseBytes responseJSONBytes
	err = json.Unmarshal(responseBytes, &parsedResponseBytes)
	if err != nil {
		return "", err
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		return "", logger.Errorf("Unexpected status code: %s | %s", strconv.Itoa(response.StatusCode), string(responseBytes))
	}

	VPS := Instance{
		Creation_Time: parsedResponseBytes.Creation_Time,
		Host_UUID:     parsedResponseBytes.Host_UUID,
		Id:            parsedResponseBytes.Id,
		Host_Image:    parsedResponseBytes.Host_Image,
		Ipv4:          parsedResponseBytes.Ipv4,
		Ipv6:          parsedResponseBytes.Ipv6,
		Label:         parsedResponseBytes.Label,
		Region:        parsedResponseBytes.Region,
		Type:          parsedResponseBytes.Type,
	}

	fmt.Printf("%v\n", parsedResponseBytes)
	fmt.Println(reflect.TypeOf(parsedResponseBytes))

	var allinstances InstancesToml

	if _, err := os.Stat(instanceFile); err == nil {
		if _, err := toml.DecodeFile(instanceFile, &allinstances); err != nil {
			logger.Error("Unable to load pre-existing instance file: " + logger.Highlight(instanceFile) + string(err.Error()))
			return "", err
		}
	} else {
		allinstances = make(InstancesToml)
	}

	allinstances[VPS.Label] = VPS

	logger.Info("Writing new updated instance file")
	f, err := os.Create(instanceFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	logger.Info("Encoding")
	err = toml.NewEncoder(f).Encode(allinstances)
	if err != nil {
		return "", err
	}

	logger.Info("Wrote VPS creation instance file: " + logger.Highlight(instanceFile))
	if err != nil {
		return "", err
	}

	// return utils.PrettyPrintJSON(responseBytes), nil
	return "", nil
}
