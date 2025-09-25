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

func ChoiceGeneric(title string, options []string) string {
	var selected string

	// Convert string options to huh options
	huhOptions := make([]huh.Option[string], len(options))
	for i, option := range options {
		huhOptions[i] = huh.NewOption(option, option)
	}

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(title).
				Options(huhOptions...).
				Value(&selected),
		),
	).Run()

	if err != nil {
		return ""
	}

	return selected
}
