package ui

import (
	"fmt"
	"strings"

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

// CreateVPSWizard creates a VPS creation wizard with all steps
func CreateVPSWizard(providerData map[string]interface{}) (map[string]string, error) {
	var provider, region, image, size string

	// Extract data from provider data map
	regionOptions, _ := providerData["regions"].([]string)
	imageOptions, _ := providerData["images"].([]string)
	sizeOptions, _ := providerData["sizes"].([]string)

	if regionOptions == nil {
		regionOptions = []string{"Loading regions..."}
	}
	if imageOptions == nil {
		imageOptions = []string{"Loading images..."}
	}
	if sizeOptions == nil {
		sizeOptions = []string{"Loading sizes..."}
	}

	steps := []WizardStep{
		{
			Key:         "provider",
			Title:       "Choose your cloud provider",
			Description: "Select the cloud provider for your VPS",
			Options:     []string{"DigitalOcean", "Linode", "Vultr"},
			Value:       &provider,
		},
		{
			Key:         "region",
			Title:       "Select datacenter region",
			Description: "Choose the geographical location for your server",
			Options:     regionOptions,
			Value:       &region,
		},
		{
			Key:         "image",
			Title:       "Choose operating system",
			Description: "Select the OS image for your server",
			Options:     imageOptions,
			Value:       &image,
		},
		{
			Key:         "size",
			Title:       "Select server specifications",
			Description: "Choose the CPU, RAM, and storage configuration",
			Options:     sizeOptions,
			Value:       &size,
		},
	}

	return CreateNavigableWizard(steps)
}

// SimpleConfirmation creates a simple yes/no confirmation dialog
func SimpleConfirmation(title, description string) (bool, error) {
	var confirmed bool

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Description(description).
				Affirmative("Yes").
				Negative("No").
				Value(&confirmed),
		),
	).Run()

	if err != nil {
		return false, err
	}

	return confirmed, nil
}

// FormatOptionsForDisplay formats options for better display in the wizard
func FormatOptionsForDisplay(options []string, maxLength int) []string {
	formatted := make([]string, len(options))
	for i, opt := range options {
		if len(opt) > maxLength {
			formatted[i] = opt[:maxLength-3] + "..."
		} else {
			formatted[i] = opt
		}
	}
	return formatted
}

// SplitSelectionValue splits a formatted selection value back to its components
func SplitSelectionValue(selection string) []string {
	return strings.Fields(selection)
}

// ParseVPSSelection parses a VPS selection string into its components
func ParseVPSSelection(selection string) (id, name, ip, region string) {
	parts := strings.Split(selection, " - ")
	if len(parts) >= 4 {
		// Format: "timestamp - id - name - ip - region"
		if len(parts) >= 5 {
			return parts[1], parts[2], parts[3], parts[4]
		}
		// Format: "id - name - ip - region"
		return parts[0], parts[1], parts[2], parts[3]
	}
	return "", "", "", ""
}
