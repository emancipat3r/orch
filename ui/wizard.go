package ui

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

// WizardStep represents a step in the wizard
type WizardStep struct {
	Key         string
	Title       string
	Description string
	Options     []string
	Value       *string
}

// CreateNavigableWizard creates a wizard with full navigation support using left/right arrow keys
func CreateNavigableWizard(steps []WizardStep) (map[string]string, error) {
	if len(steps) == 0 {
		return nil, fmt.Errorf("no steps provided")
	}

	groups := make([]*huh.Group, 0, len(steps))

	for _, step := range steps {
		if len(step.Options) == 0 {
			// Skip steps with no options
			continue
		}

		options := make([]huh.Option[string], len(step.Options))
		for j, opt := range step.Options {
			options[j] = huh.NewOption(opt, opt)
		}

		description := step.Description

		group := huh.NewGroup(
			huh.NewSelect[string]().
				Key(step.Key).
				Title(step.Title).
				Description(description).
				Options(options...).
				Value(step.Value),
		).Title("")

		groups = append(groups, group)
	}

	if len(groups) == 0 {
		return nil, fmt.Errorf("no valid steps to display")
	}

	form := huh.NewForm(groups...).
		WithWidth(80).
		WithShowHelp(true).
		WithShowErrors(true)

	err := form.Run()
	if err != nil {
		return nil, err
	}

	results := make(map[string]string)
	for _, step := range steps {
		if step.Value != nil {
			results[step.Key] = *step.Value
		}
	}

	return results, nil
}
