package ui

import (
	"github.com/charmbracelet/huh"
)

func Confirm(prompt string) bool {
	var confirmed bool

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(prompt).
				Affirmative("Yes").
				Negative("No").
				Value(&confirmed),
		),
	).Run()

	if err != nil {
		return false
	}

	return confirmed
}
