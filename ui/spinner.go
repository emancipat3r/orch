package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	messageStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

type SpinnerModel struct {
	spinner  spinner.Model
	message  string
	done     bool
	success  bool
	finalMsg string
	quitting bool
}

type TickMsg time.Time
type DoneMsg struct {
	Success bool
	Message string
}

type UpdateMsg string

func (m SpinnerModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tea.Every(time.Second, func(t time.Time) tea.Msg {
			return TickMsg(t)
		}),
	)
}

func (m SpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}

	case TickMsg:
		if !m.done {
			return m, tea.Every(time.Second, func(t time.Time) tea.Msg {
				return TickMsg(t)
			})
		}

	case UpdateMsg:
		m.message = string(msg)
		return m, nil

	case DoneMsg:
		m.done = true
		m.success = msg.Success
		m.finalMsg = msg.Message
		return m, tea.Quit

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m SpinnerModel) View() string {
	if m.quitting {
		return "\n"
	}

	if m.done {
		if m.success {
			if m.finalMsg == "" {
				return ""
			}
			return fmt.Sprintf("✓ %s\n", m.finalMsg)
		}
		return fmt.Sprintf("✗ %s\n", m.finalMsg)
	}

	str := fmt.Sprintf("%s %s",
		spinnerStyle.Render(m.spinner.View()),
		messageStyle.Render(m.message))
	return str
}

func NewSpinnerModel(message string) SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	return SpinnerModel{
		spinner: s,
		message: message,
	}
}

// IPWaitSpinner creates and runs a spinner for IP waiting
func IPWaitSpinner(ctx context.Context, message string) (*tea.Program, chan DoneMsg) {
	model := NewSpinnerModel(message)
	p := tea.NewProgram(model)

	doneChan := make(chan DoneMsg, 1)

	// Start the program in a goroutine
	go func() {
		if _, err := p.Run(); err != nil {
			doneChan <- DoneMsg{Success: false, Message: "Spinner error: " + err.Error()}
		}
	}()

	// Handle context cancellation
	go func() {
		<-ctx.Done()
		p.Send(DoneMsg{Success: false, Message: "Operation cancelled"})
	}()

	return p, doneChan
}

// UpdateSpinnerMessage updates the spinner message
func UpdateSpinnerMessage(p *tea.Program, message string) {
	p.Send(UpdateMsg(message))
}

// FinishSpinner completes the spinner with success/failure
func FinishSpinner(p *tea.Program, success bool, message string) {
	p.Send(DoneMsg{Success: success, Message: message})
}
