package ui

import (
	"github.com/charmbracelet/huh"
)

func ChoiceProvider() string {
	var provider string

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select your provider").
				Options(
					huh.NewOption("DigitalOcean", "DigitalOcean"),
					huh.NewOption("Linode", "Linode"),
					huh.NewOption("Vultr", "Vultr"),
				).
				Value(&provider),
		),
	).Run()

	if err != nil {
		return ""
	}

	return provider
}
