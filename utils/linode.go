package utils

import (
	"io"
	"net/http"
	"strconv"

	"github.com/emancipat3r/vps3/logger"
)

func GetLinodeAPIKey(configFile string, provider string) string {

	providerKey := ParseCreds(configFile, provider)
	if providerKey == "" {
		logger.Error("Failed to parse provider credentials")
		return ""
	}
	return providerKey
}

func GetLinodeAccount(providerKey string) (string, error) {
	if providerKey == "" {
		return "", logger.Errorf("Provider key is empty")
	}

	request, err := http.NewRequest("GET", "https://api.linode.com/v4/account", nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+providerKey)

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

	formattedResponse := PrettyPrintJSON(responseBytes)
	return formattedResponse, nil
}

func ListLinodes(providerKey string) (string, error) {
	if providerKey == "" {
		return "", logger.Errorf("Provider key is empty")
	}

	request, err := http.NewRequest("GET", "https://api.linode.com/v4/linode/instances?page=1&page_size=100", nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+providerKey)

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

	formattedResponse := PrettyPrintJSON(responseBytes)
	return formattedResponse, nil
}
