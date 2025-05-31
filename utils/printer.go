package utils

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	//timestampStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	timestampStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("247"))
	bracketStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Bold(true)  // White bold
	infoStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true) // Blue
	debugStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("247")).Italic(true)
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	warnStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
)

func timestamp() string {
	return timestampStyle.Render(time.Now().Format("15:04:05"))
}

func bracketed(label string, style lipgloss.Style) string {
	return bracketStyle.Render("[") + style.Render(label) + bracketStyle.Render("]")
}

func Info(msg string) {
	fmt.Printf("%s %s  %s\n", timestamp(), bracketed("INFO", infoStyle), msg)
}

func Debug(msg string) {
	fmt.Printf("%s %s %s\n", timestamp(), bracketed("DEBUG", debugStyle), msg)
}

func Error(msg string) {
	fmt.Printf("%s %s %s\n", timestamp(), bracketed("ERROR", errorStyle), msg)
}

func Warn(msg string) {
	fmt.Printf("%s %s  %s\n", timestamp(), bracketed("WARN", warnStyle), msg)
}
