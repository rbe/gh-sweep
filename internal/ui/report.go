package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rbe/gh-sweep/internal/format"
	"github.com/rbe/gh-sweep/internal/project"
)

// reportState is the report screen: a title, a note above the lines, the
// lines of one report, the first line shown, and the screen to return to.
type reportState struct {
	title  string
	note   string
	lines  []string
	scroll int
	from   screen
}

// openUsage shows the storage usage breakdown of the scanned projects.
// A project that is not scanned has no numbers, so the note counts it.
func (m *model) openUsage() {
	var scanned []project.Project
	unscanned := 0
	for _, p := range m.projects {
		if p.Scanned {
			scanned = append(scanned, p)
		} else {
			unscanned++
		}
	}
	note := ""
	if unscanned > 0 {
		note = fmt.Sprintf("%d projects are not scanned and have no line. Press s on the project list to scan them.", unscanned)
	}
	m.showReport("storage usage", note, project.UsageLines(scanned))
}

// openArtifactList shows one line for each artifact of the open project.
func (m *model) openArtifactList() {
	p := m.projects[m.open]
	m.showReport("artifacts of "+p.NWO, "Dates are UTC.", project.ArtifactLines([]project.Project{p}))
}

func (m *model) showReport(title, note string, lines []string) {
	m.report = reportState{title: title, note: note, lines: lines, from: m.screen}
	m.screen = scrReport
}

// onReport scrolls the report, goes back to the screen it came from, or quits.
func (m model) onReport(s string) (tea.Model, tea.Cmd) {
	switch s {
	case "q":
		return m, tea.Quit
	case "up", "k":
		m.scrollReport(-1)
	case "down", "j":
		m.scrollReport(1)
	case "b", "esc", "left", "h":
		m.screen = m.report.from
	}
	return m, nil
}

func (m *model) scrollReport(d int) {
	last := len(m.report.lines) - m.reportHeight()
	if last < 0 {
		last = 0
	}
	m.report.scroll += d
	if m.report.scroll > last {
		m.report.scroll = last
	}
	if m.report.scroll < 0 {
		m.report.scroll = 0
	}
}

// reportHeight is the number of report lines that fit between the header
// and the key line.
func (m model) reportHeight() int {
	h := m.height - 6
	if h < 1 {
		h = 1
	}
	return h
}

func (m model) viewReport() string {
	r := m.report
	var b strings.Builder
	b.WriteString(titleStyle.Render("gh-sweep") + nameStyle.Render("  "+r.title) + "\n")
	if r.note != "" {
		b.WriteString(warnStyle.Render(format.Fit(r.note, m.width-1)))
	}
	b.WriteString("\n\n")
	end := r.scroll + m.reportHeight()
	if end > len(r.lines) {
		end = len(r.lines)
	}
	for _, line := range r.lines[r.scroll:end] {
		b.WriteString("  " + format.Fit(line, m.width-3) + "\n")
	}
	b.WriteByte('\n')
	b.WriteString(dimStyle.Render(format.Fit(fmt.Sprintf("lines %d-%d of %d    j/k scroll    b back    q quit", r.scroll+1, end, len(r.lines)), m.width-1)) + "\n")
	return b.String()
}
