package utils

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	ti   textinput.Model
	done bool
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			m.done = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.done {
		return ""
	}
	return m.ti.View()
}

// Prompt displays an input prompt and returns the user's input.
func Prompt(placeholder string) string {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()

	m := model{ti: ti}
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr)) // optional: stderr avoids stdout clashes
	if err := p.Start(); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	return m.ti.Value()
}
