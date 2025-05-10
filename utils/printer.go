package utils

import {
	"fmt"
	"github.com/charmbracelet/lipgloss"
}

// global style
var style = lipgloss.NewStyle(),
    Foreground(lipgloss.Color("205")),
    Padding(0, 1),
    Bold(true)

func LgPrint(msg string) {
fmt.Println(style.Render(msg))
}
