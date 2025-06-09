package ui

import (
	"github.com/charmbracelet/huh"
	"github.com/emancipat3r/vps3/logger"
)

func ChoiceProvider() string {
	var provider string

	huh.NewSelect[string]().
		Title("Select your provider").
		Options(
			huh.NewOption("DigitalOcean", "DigitalOcean"),
			huh.NewOption("Linode", "Linode"),
			huh.NewOption("Vultr", "Vultr"),
		).
		Value(&provider).
		Run()

	logger.Info("You selected: " + provider)
	return provider
}
