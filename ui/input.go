package ui

import (
	"github.com/charmbracelet/huh"
)

func Input(prompt string) string {
	var val string

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(prompt).
				Value(&val),
		),
	).Run()

	if err != nil {
		return "" // or panic/log/fallback
	}

	return val
}
