package ui

import (
	"github.com/charmbracelet/huh"
)

func Select(prompt string, options []string) string {
	var choice string

	items := make([]huh.Option[string], len(options))
	for i, opt := range options {
		items[i] = huh.NewOption(opt, opt)
	}

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(prompt).
				Options(items...).
				Value(&choice),
		),
	).Run()

	if err != nil {
		return ""
	}

	return choice
}
