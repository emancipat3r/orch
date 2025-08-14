package providers

import (
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
