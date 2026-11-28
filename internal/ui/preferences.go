package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/rbe/gh-sweep/internal/project"
)

// preferencesState is the preferences screen: one field for each value of
// project.Preferences, and the field that has the focus.
type preferencesState struct {
	fields []filterField
	focus  int
}

// openPreferences shows the preferences in force, one field each.
func (m model) openPreferences() (model, tea.Cmd) {
	m.prefs = preferencesState{}
	for _, f := range project.Prefs().Fields() {
		in := newInput(f.Text)
		in.SetValue(f.Text)
		in.Blur()
		m.prefs.fields = append(m.prefs.fields, filterField{name: f.Key, input: in})
	}
	m.prefs.fields[0].input.Focus()
	m.screen = scrPreferences
	return m, textinput.Blink
}

// onPreferences moves between the fields, saves with enter, and leaves
// without a change with esc.
func (m model) onPreferences(s string) (tea.Model, tea.Cmd) {
	m.status = ""
	switch s {
	case "esc":
		m.screen = scrProjects
		return m, nil
	case "tab", "down":
		m.focusPreference(m.prefs.focus + 1)
		return m, nil
	case "shift+tab", "up":
		m.focusPreference(m.prefs.focus - 1)
		return m, nil
	case "enter":
		return m.savePreferences()
	}
	f := &m.prefs.fields[m.prefs.focus]
	var cmd tea.Cmd
	f.input, cmd = f.input.Update(tea.KeyMsg(keyFromString(s)))
	return m, cmd
}

func (m *model) focusPreference(i int) {
	n := len(m.prefs.fields)
	m.prefs.fields[m.prefs.focus].input.Blur()
	m.prefs.focus = (i + n) % n
	m.prefs.fields[m.prefs.focus].input.Focus()
}

// savePreferences reads every field, puts the values in force, and writes
// them to the file of the options. A wrong value keeps the screen open and
// changes nothing.
func (m model) savePreferences() (tea.Model, tea.Cmd) {
	p := project.Prefs()
	for _, f := range m.prefs.fields {
		var err error
		if p, err = p.WithField(f.name, strings.TrimSpace(f.input.Value())); err != nil {
			m.status = err.Error()
			return m, nil
		}
	}
	if err := p.Check(); err != nil {
		m.status = err.Error()
		return m, nil
	}
	if m.opts.PreferencesPath != "" {
		if err := project.SavePreferences(m.opts.PreferencesPath, p); err != nil {
			m.status = "Could not save the preferences: " + err.Error()
			return m, nil
		}
	}
	if err := project.UsePreferences(p); err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.screen = scrProjects
	m.status = "Saved the preferences"
	if m.opts.PreferencesPath == "" {
		m.status = "The preferences apply until gh-sweep quits"
	}
	return m, nil
}

// viewPreferences draws one line for each field, the file, and the keys.
func (m model) viewPreferences() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("gh-sweep") + "  " + nameStyle.Render("Preferences") + "\n\n")
	labels := map[string]string{}
	for _, f := range project.Prefs().Fields() {
		labels[f.Key] = f.Label
	}
	for i, f := range m.prefs.fields {
		b.WriteString(pointer(i == m.prefs.focus) + fmt.Sprintf("%-26s", labels[f.name]) + f.input.View() + "\n")
	}
	b.WriteString("\n" + dimStyle.Render("A size is bytes, or a number with KB, MB, or GB.") + "\n")
	if m.opts.PreferencesPath != "" {
		b.WriteString(dimStyle.Render("File: "+m.opts.PreferencesPath) + "\n")
	}
	if m.status != "" {
		b.WriteString("\n" + warnStyle.Render(m.status) + "\n")
	}
	b.WriteString("\n" + dimStyle.Render("tab next field    enter save    esc back") + "\n")
	return b.String()
}
