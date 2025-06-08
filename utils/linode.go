package utils

import (
	"github.com/emancipat3r/vps3/logger"
)

func GetLinodeAPIKey(configFile string, provider string) string {

	providerKey := ParseCreds(configFile, provider)
	if providerKey != "" {
		logger.Info("Provider key: " + providerKey)
	} else {
		logger.Error("Failed to parse provider credentials")
		return ""
	}
	return providerKey
}
