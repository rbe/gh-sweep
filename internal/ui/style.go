package ui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	nameStyle    = lipgloss.NewStyle().Bold(true)
	checkOn      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	checkOff     = lipgloss.NewStyle().Faint(true)
	checkSome    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))
	dimStyle     = lipgloss.NewStyle().Faint(true)
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	errStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("1"))
	okStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	cursorStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	sectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
)

func styledBox(state string) string {
	switch state {
	case "all":
		return checkOn.Render("[x]")
	case "some":
		return checkSome.Render("[-]")
	default:
		return checkOff.Render("[ ]")
	}
}
