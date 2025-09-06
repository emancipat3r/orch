package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

func InstanceTable(rows [][]string) string {
	var (
		purple = lipgloss.Color("99")
		gray   = lipgloss.Color("245")

		headerStyle = lipgloss.NewStyle().Foreground(purple).Bold(true).Align(lipgloss.Center)
		cellStyle   = lipgloss.NewStyle().Padding(0, 1).Width(14)
		rowStyle    = cellStyle.Foreground(gray)
	)

	t := table.New()
	//t.Border(lipgloss.NormalBorder())
	t.Border(lipgloss.ASCIIBorder())
	t.BorderStyle(lipgloss.NewStyle().Foreground(purple))
	t.BorderRow(true)
	t.Headers("ID", "IPv4", "Region", "Image", "Type", "Creation Time", "Status")
	t.StyleFunc(func(row, col int) lipgloss.Style {
		switch {
		case row == table.HeaderRow:
			return headerStyle
		default:
			return rowStyle
		}
	})

	t.Rows(rows...)

	return t.Render()
}
