package logger

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	timestampStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	infoStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	debugStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	warnStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	successStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	highlightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
)

func log(style lipgloss.Style, label, msg string) {
	ts := timestampStyle.Render(time.Now().Format("15:04:05"))
	fmt.Printf("%s %s %s\n", ts, style.Render("["+label+"]"), msg)
}

// Errorf logs an error and returns it. Unlike fmt.Sprintf, fmt.Errorf
// honors %w, so callers can chain wrapped errors and errors.Is/errors.As
// works on the result.
func Errorf(format string, args ...interface{}) error {
	err := fmt.Errorf(format, args...)
	log(errorStyle, "ERROR", err.Error())
	return err
}

func Highlight(text string) string {
	return highlightStyle.Render(text)
}

func Info(msg string)    { log(infoStyle, "INFO", msg) }
func Debug(msg string)   { log(debugStyle, "DEBUG", msg) }
func Warn(msg string)    { log(warnStyle, "WARN", msg) }
func Error(msg string)   { log(errorStyle, "ERROR", msg) }
func Success(msg string) { log(successStyle, "OK", msg) }
