package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/rbe/gh-sweep/internal/format"
	"github.com/rbe/gh-sweep/internal/project"
	"github.com/rbe/gh-sweep/internal/sweep"
)

// actionFlags are the write flags that the action screen offers, in the
// order of its menu.
var actionFlags = []string{"--delete-runs", "--delete-run-logs", "--delete-releases", "--delete-release-assets"}

// stage is one step of the action screen.
type stage int

const (
	stageMenu stage = iota
	stageForm
	stageListing
	stagePreview
	stageConfirm
	stageApplying
)

// actionRow is one entry of the preview: the repository, the entry, and
// whether it is checked. A row with no write cannot be checked.
type actionRow struct {
	nwo   string
	entry sweep.Entry
	on    bool
}

// filterField is one text input of the filter form.
type filterField struct {
	name  string
	input textinput.Model
}

// actionState is the action screen: the menu, the chosen action, its
// filter form, the repositories it runs in, and the rows of the preview.
type actionState struct {
	stage   stage
	menu    []sweep.Action
	pick    int
	action  sweep.Action
	fields  []filterField
	focus   int
	filter  project.Filter
	targets []project.Project
	rows    []actionRow
	errs    []string
	cursor  int
	scroll  int
	token   string
	input   textinput.Model
	next    int
	done    []string
	total   project.Total
	stopped bool
}

// listEntries reads the entries of an action in one repository. A test
// replaces it.
var listEntries = func(a sweep.Action, p project.Project, f project.Filter) ([]sweep.Entry, error) {
	return a.List(p, f, time.Now())
}

// actionsListedMsg carries the rows of all targets and one error line for
// each repository that could not be read.
type actionsListedMsg struct {
	rows []actionRow
	errs []string
}

// actionAppliedMsg is the result of the write of one row.
type actionAppliedMsg struct {
	err error
}

// openActions shows the menu of actions for the marked projects, or for
// the project under the cursor when none is marked.
func (m model) openActions() (model, tea.Cmd) {
	targets := m.actionTargets()
	if len(targets) == 0 {
		m.status = "No project to run an action in"
		return m, nil
	}
	var menu []sweep.Action
	for _, flag := range actionFlags {
		if a, ok := sweep.ByFlag(flag); ok {
			menu = append(menu, a)
		}
	}
	m.act = actionState{stage: stageMenu, menu: menu, targets: targets}
	m.screen = scrAction
	return m, nil
}

// actionTargets are the marked projects, or the project under the cursor.
func (m model) actionTargets() []project.Project {
	var marked []project.Project
	for _, p := range m.projects {
		if p.On {
			marked = append(marked, p)
		}
	}
	if len(marked) > 0 {
		return marked
	}
	if m.cursor < len(m.projView) {
		return []project.Project{m.projects[m.projView[m.cursor]]}
	}
	return nil
}

// onAction routes a key to the current stage of the action screen.
func (m model) onAction(s string) (tea.Model, tea.Cmd) {
	switch m.act.stage {
	case stageMenu:
		return m.onActionMenu(s)
	case stageForm:
		return m.onActionForm(s)
	case stagePreview:
		return m.onActionPreview(s)
	case stageConfirm:
		return m.onActionConfirm(s)
	}
	return m, nil
}

func (m model) onActionMenu(s string) (tea.Model, tea.Cmd) {
	m.status = ""
	switch s {
	case "q":
		return m, tea.Quit
	case "esc", "b":
		m.screen = scrProjects
	case "up", "k":
		if m.act.pick > 0 {
			m.act.pick--
		}
	case "down", "j":
		if m.act.pick < len(m.act.menu)-1 {
			m.act.pick++
		}
	case "enter":
		return m.chooseAction(m.act.menu[m.act.pick])
	}
	return m, nil
}

// chooseAction opens the filter form of the action, or lists its entries
// when it reads no filter.
func (m model) chooseAction(a sweep.Action) (model, tea.Cmd) {
	m.act.action = a
	m.act.fields = nil
	m.act.focus = 0
	for _, name := range a.Filters {
		in := newInput(sweep.FilterLabels[name])
		in.Blur()
		if name == "package-type" {
			in.SetValue("container")
		}
		m.act.fields = append(m.act.fields, filterField{name: name, input: in})
	}
	if len(m.act.fields) == 0 {
		return m.listAction()
	}
	m.act.fields[0].input.Focus()
	m.act.stage = stageForm
	return m, textinput.Blink
}

func (m model) onActionForm(s string) (tea.Model, tea.Cmd) {
	m.status = ""
	switch s {
	case "esc":
		m.act.stage = stageMenu
		return m, nil
	case "tab", "down":
		m.focusField(m.act.focus + 1)
		return m, nil
	case "shift+tab", "up":
		m.focusField(m.act.focus - 1)
		return m, nil
	case "enter":
		return m.listAction()
	}
	f := &m.act.fields[m.act.focus]
	var cmd tea.Cmd
	f.input, cmd = f.input.Update(tea.KeyMsg(keyFromString(s)))
	return m, cmd
}

func (m *model) focusField(i int) {
	n := len(m.act.fields)
	m.act.fields[m.act.focus].input.Blur()
	m.act.focus = (i + n) % n
	m.act.fields[m.act.focus].input.Focus()
}

// listAction parses the form, checks the filters with the rules of the
// command line, and reads the entries of each target.
func (m model) listAction() (model, tea.Cmd) {
	var f project.Filter
	var given []string
	for _, field := range m.act.fields {
		v := strings.TrimSpace(field.input.Value())
		if v == "" {
			continue
		}
		if err := sweep.SetFilter(&f, field.name, v); err != nil {
			m.status = sweep.FilterLabels[field.name] + ": " + err.Error()
			return m, nil
		}
		if field.name != "package-type" || v != "container" {
			given = append(given, field.name)
		}
	}
	if f.PackageType == "" {
		f.PackageType = "container"
	}
	var names []string
	for _, p := range m.act.targets {
		names = append(names, p.NWO)
	}
	if err := m.act.action.CheckFilters(given, f, sweep.Scope{Repos: names}); err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.act.filter = f
	m.act.stage = stageListing
	a, targets := m.act.action, m.act.targets
	return m, func() tea.Msg {
		var msg actionsListedMsg
		for _, p := range targets {
			entries, err := listEntries(a, p, f)
			for _, e := range entries {
				msg.rows = append(msg.rows, actionRow{nwo: p.NWO, entry: e, on: e.Apply != nil})
			}
			if err != nil {
				msg.errs = append(msg.errs, p.NWO+": "+err.Error())
			}
		}
		return msg
	}
}

// onActionsListed shows the preview of the rows.
func (m model) onActionsListed(msg actionsListedMsg) (tea.Model, tea.Cmd) {
	m.act.rows = msg.rows
	m.act.errs = msg.errs
	m.act.cursor = 0
	m.act.scroll = 0
	m.act.stage = stagePreview
	return m, nil
}

func (m model) onActionPreview(s string) (tea.Model, tea.Cmd) {
	m.status = ""
	switch s {
	case "q":
		return m, tea.Quit
	case "esc", "b":
		if len(m.act.fields) > 0 {
			m.act.stage = stageForm
		} else {
			m.act.stage = stageMenu
		}
	case "up", "k":
		m.moveActionCursor(-1)
	case "down", "j":
		m.moveActionCursor(1)
	case " ":
		if r := m.currentRow(); r != nil && r.entry.Apply != nil {
			r.on = !r.on
		}
	case "a", "n":
		for i := range m.act.rows {
			if m.act.rows[i].entry.Apply != nil {
				m.act.rows[i].on = s == "a"
			}
		}
	case "d", "enter":
		return m.confirmAction()
	}
	return m, nil
}

func (m *model) currentRow() *actionRow {
	if m.act.cursor < len(m.act.rows) {
		return &m.act.rows[m.act.cursor]
	}
	return nil
}

func (m *model) moveActionCursor(d int) {
	c := m.act.cursor + d
	if c < 0 || c >= len(m.act.rows) {
		return
	}
	m.act.cursor = c
	h := m.listH()
	if c < m.act.scroll {
		m.act.scroll = c
	}
	if c >= m.act.scroll+h {
		m.act.scroll = c - h + 1
	}
}

// checked counts the checked rows, their bytes, and their repositories.
func (m model) checked() (int, int64, int) {
	n := 0
	var bytes int64
	repos := map[string]bool{}
	for _, r := range m.act.rows {
		if r.on {
			n++
			bytes += r.entry.Total.Bytes
			repos[r.nwo] = true
		}
	}
	return n, bytes, len(repos)
}

// confirmAction asks before the writes. More than strictCount rows, a
// gigabyte or more, or more than one repository needs a typed word.
func (m model) confirmAction() (model, tea.Cmd) {
	n, bytes, repos := m.checked()
	if n == 0 {
		m.status = "Nothing checked"
		return m, nil
	}
	m.act.token = ""
	if n > strictCount || bytes >= strictBytes || repos > 1 {
		m.act.token = m.act.action.Verb
		m.act.input = newInput(m.act.token)
	}
	m.act.stage = stageConfirm
	if m.act.token != "" {
		return m, textinput.Blink
	}
	return m, nil
}

func (m model) onActionConfirm(s string) (tea.Model, tea.Cmd) {
	m.status = ""
	if m.act.token != "" {
		switch s {
		case "enter":
			if strings.TrimSpace(m.act.input.Value()) == m.act.token {
				return m.applyAction()
			}
			m.status = "Type " + m.act.token + " to confirm"
		case "esc":
			m.act.stage = stagePreview
		default:
			var cmd tea.Cmd
			m.act.input, cmd = m.act.input.Update(tea.KeyMsg(keyFromString(s)))
			return m, cmd
		}
		return m, nil
	}
	switch s {
	case "y", "Y":
		return m.applyAction()
	case "n", "N", "esc", "q", "enter":
		m.act.stage = stagePreview
	}
	return m, nil
}

// applyAction starts the writes of the checked rows, one at a time.
func (m model) applyAction() (model, tea.Cmd) {
	m.act.next = -1
	m.act.done = nil
	m.act.total = project.Total{}
	m.act.stopped = false
	m.act.stage = stageApplying
	return m.applyNext(nil)
}

// applyNext records the result of the last write and starts the next
// checked row. With --dry-run, no row writes. After the last row, or after
// ctrl-c, the result report opens.
func (m model) applyNext(err error) (model, tea.Cmd) {
	if m.act.next >= 0 {
		r := m.act.rows[m.act.next]
		if err != nil {
			m.act.errs = append(m.act.errs, r.nwo+": "+err.Error())
		} else {
			m.act.done = append(m.act.done, r.entry.Line(m.opts.DryRun))
			m.act.total = m.act.total.Add(r.entry.Total)
		}
	}
	for m.act.next++; m.act.next < len(m.act.rows) && !m.act.rows[m.act.next].on; m.act.next++ {
	}
	if m.act.stopped || m.act.next >= len(m.act.rows) {
		return m.finishAction(), nil
	}
	r := m.act.rows[m.act.next]
	if m.opts.DryRun {
		return m.applyNext(nil)
	}
	return m, func() tea.Msg { return actionAppliedMsg{err: r.entry.Apply()} }
}

// finishAction shows the lines of the done rows, the total line, and the
// failed rows in the report screen.
func (m model) finishAction() model {
	a := m.act.action
	lines := append([]string{}, m.act.done...)
	lines = append(lines, a.Total(m.act.total, m.opts.DryRun))
	for _, e := range m.act.errs {
		lines = append(lines, "failed: "+e)
	}
	note := ""
	switch {
	case m.act.stopped:
		note = "Stopped by ctrl-c. The rows after the last line did not run."
	case m.opts.DryRun:
		note = "Dry run: no request changed GitHub."
	}
	m.screen = scrProjects
	m.showReport(a.Flag, note, lines)
	return m
}

func (m model) viewAction() string {
	var b strings.Builder
	head := "gh-sweep  actions"
	if m.act.action.Flag != "" && m.act.stage != stageMenu {
		head += "  " + m.act.action.Flag
	}
	if m.opts.DryRun {
		head += "   dry-run"
	}
	b.WriteString(titleStyle.Render(head) + "\n")
	b.WriteString(dimStyle.Render(format.Fit("in "+m.targetNames(), m.width-1)) + "\n\n")
	switch m.act.stage {
	case stageMenu:
		for i, a := range m.act.menu {
			b.WriteString(pointer(i == m.act.pick) + a.Title + dimStyle.Render("  "+a.Flag) + "\n")
		}
		b.WriteString("\n" + dimStyle.Render("j/k move    enter choose    b back    q quit") + "\n")
	case stageForm:
		b.WriteString(nameStyle.Render(m.act.action.Title) + "\n\n")
		for i, f := range m.act.fields {
			b.WriteString(pointer(i == m.act.focus) + fmt.Sprintf("%-14s", f.name) + f.input.View() + "\n")
		}
		b.WriteString("\n" + dimStyle.Render("Leave a field empty to not use it.") + "\n")
		b.WriteString(dimStyle.Render("tab next field    enter show the entries    esc back") + "\n")
	case stageListing:
		b.WriteString("Reading the entries…\n")
	case stagePreview:
		m.viewActionRows(&b)
	case stageConfirm:
		m.viewActionConfirm(&b)
	case stageApplying:
		b.WriteString(fmt.Sprintf("Running %d of %d\n\n", len(m.act.done)+len(m.act.errs)+1, len(m.act.rows)))
		b.WriteString(dimStyle.Render("ctrl-c stops after the current request") + "\n")
	}
	if m.status != "" {
		b.WriteString("\n" + warnStyle.Render(format.Fit(m.status, m.width-1)) + "\n")
	}
	return b.String()
}

func (m model) viewActionRows(b *strings.Builder) {
	if len(m.act.rows) == 0 {
		b.WriteString("The filters select no entry.\n")
	}
	end := m.act.scroll + m.listH()
	if end > len(m.act.rows) {
		end = len(m.act.rows)
	}
	for i := m.act.scroll; i < end; i++ {
		r := m.act.rows[i]
		box := "    "
		if r.entry.Apply != nil {
			state := "none"
			if r.on {
				state = "all"
			}
			box = styledBox(state) + " "
		}
		b.WriteString(pointer(i == m.act.cursor) + box + format.Fit(r.entry.Line(true), m.width-9) + "\n")
	}
	for _, e := range m.act.errs {
		b.WriteString(errStyle.Render(format.Fit("could not read "+e, m.width-1)) + "\n")
	}
	n, bytes, _ := m.checked()
	b.WriteString(fmt.Sprintf("\nchecked %d of %d    %s\n", n, len(m.act.rows), format.HumanSize(bytes)))
	b.WriteString(dimStyle.Render("space toggle    a all    n none    d run the checked rows    b back    q quit") + "\n")
}

func (m model) viewActionConfirm(b *strings.Builder) {
	n, bytes, repos := m.checked()
	a := m.act.action
	b.WriteString(errStyle.Render(fmt.Sprintf("%s %d %s in %d repositories", a.Verb, n, a.Noun, repos)) + "\n\n")
	if m.opts.DryRun {
		b.WriteString(warnStyle.Render("Dry run: no request will change GitHub.") + "\n\n")
	} else {
		b.WriteString("GitHub cannot undo this.\n")
		if bytes > 0 {
			b.WriteString("size " + format.HumanSize(bytes) + "\n")
		}
		b.WriteByte('\n')
	}
	if m.act.token != "" {
		b.WriteString("Type " + m.act.token + " to confirm\n")
		b.WriteString(m.act.input.View() + "\n")
		b.WriteString(dimStyle.Render("enter confirms    esc goes back") + "\n")
		return
	}
	b.WriteString("y   run now\n")
	b.WriteString(dimStyle.Render("n   go back") + "\n")
}

func (m model) targetNames() string {
	var names []string
	for _, p := range m.act.targets {
		names = append(names, p.NWO)
	}
	return strings.Join(names, ", ")
}

func pointer(on bool) string {
	if on {
		return cursorStyle.Render("> ")
	}
	return "  "
}
