// Package ui is the Bubble Tea checklist.
package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/rbe/gh-sweep/internal/format"
	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
)

type screen int

const (
	scrProjects screen = iota
	scrItems
	scrConfirm
	scrWorking
	scrResult
	scrHelp
	scrPrompt
	scrReport
	scrAction
	scrPreferences
)

// Options is the checklist the command line asked for.
type Options struct {
	Repos     []string
	Org       string
	User      string
	Mine      bool
	WantArt   bool
	WantCache bool
	DryRun    bool
	Filter    string
	// PreferencesPath is the file that the preferences screen writes. An
	// empty path keeps a change until gh-sweep quits.
	PreferencesPath string
}

type delJob struct {
	proj     int
	id       string
	kind     string
	name     string
	nwo      string
	size     int64
	apiHost  string
	owner    string
	repoName string
}

type confirmState struct {
	scope   string
	strict  bool
	token   string
	nwo     string
	count   int
	bytes   int64
	expired int64
	lines   []string
}

type model struct {
	opts    Options
	prefs   preferencesState
	screen  screen
	width   int
	height  int
	ready   bool
	loadErr string

	projects []project.Project
	projView []int
	open     int
	rows     []project.ViewRow
	cursor   int
	scroll   int

	filter         string
	filterEdit     bool
	projFilter     string
	projFilterEdit bool
	status         string
	note           string
	busy           string
	pendingOpen    int
	pendingWipe    bool

	confirm    confirmState
	input      textinput.Model
	promptKind string
	promptFrom screen
	helpFrom   screen

	report reportState
	act    actionState

	jobs        []delJob
	jobi        int
	freed       int64
	failLines   []string
	fails       int
	okArt       int
	okCache     int
	stop        bool
	resultWhere string

	sessionArt   int
	sessionCache int
	sessionBytes int64
	sessionRepos string
}

// Run starts the checklist and prints a one-line summary when anything was deleted.
func Run(o Options) error {
	prog := tea.NewProgram(newModel(o), tea.WithAltScreen())
	final, err := prog.Run()
	if err != nil {
		return err
	}
	if m, ok := final.(model); ok {
		if line := m.summary(); line != "" {
			fmt.Println(line)
		}
	}
	return nil
}

func newModel(o Options) model {
	m := model{
		opts:        o,
		screen:      scrProjects,
		width:       80,
		height:      24,
		open:        -1,
		pendingOpen: -1,
		filter:      o.Filter,
	}
	return m
}

func (m model) summary() string {
	total := m.sessionArt + m.sessionCache
	if total == 0 {
		return ""
	}
	verb := "deleted"
	if m.opts.DryRun {
		verb = "would delete"
	}
	where := m.sessionRepos
	if where == "" {
		where = "GitHub"
	}
	return fmt.Sprintf("gh-sweep: %s %d %s and %d %s from %s (%s)",
		verb,
		m.sessionArt, plural(m.sessionArt, "artifact", "artifacts"),
		m.sessionCache, plural(m.sessionCache, "cache", "caches"),
		where, format.HumanSize(m.sessionBytes))
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func (m model) Init() tea.Cmd {
	return m.loadCmd()
}

type loadedMsg struct {
	projects []project.Project
	note     string
	err      error
}

type scannedMsg struct {
	projects []project.Project
	failed   string
}

type delMsg struct {
	ok    bool
	err   string
	label string
	done  bool
}

func (m model) loadCmd() tea.Cmd {
	o := m.opts
	return func() tea.Msg {
		if err := gh.Auth(); err != nil {
			return loadedMsg{err: err}
		}
		projects, note, err := collect(o)
		return loadedMsg{projects: projects, note: note, err: err}
	}
}

func collect(o Options) ([]project.Project, string, error) {
	var specs []string
	var note string
	switch {
	case len(o.Repos) > 0:
		specs = o.Repos
	case o.Org != "":
		names, err := gh.ListRepos(o.Org)
		if err != nil {
			return nil, "", err
		}
		specs = names
		if len(names) >= gh.ListCap {
			note = fmt.Sprintf("Showing the first %d repositories.", gh.ListCap)
		}
	case o.User != "":
		names, err := gh.ListRepos(o.User)
		if err != nil {
			return nil, "", err
		}
		specs = names
		if len(names) >= gh.ListCap {
			note = fmt.Sprintf("Showing the first %d repositories.", gh.ListCap)
		}
	default:
		names, err := gh.ListRepos("")
		if err != nil {
			return nil, "", err
		}
		specs = names
		if len(names) >= gh.ListCap {
			note = fmt.Sprintf("Showing the first %d repositories.", gh.ListCap)
		}
	}
	var projects []project.Project
	seen := map[string]bool{}
	for _, spec := range specs {
		p, err := project.Parse(spec)
		if err != nil {
			return nil, "", err
		}
		if seen[p.NWO] {
			continue
		}
		seen[p.NWO] = true
		projects = append(projects, p)
	}
	return projects, note, nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.width < 40 {
			m.width = 40
		}
		if m.height < 10 {
			m.height = 10
		}
		m.ensure(m.rowCount())
		return m, nil
	case loadedMsg:
		m.ready = true
		if msg.err != nil {
			m.loadErr = msg.err.Error()
			return m, nil
		}
		m.projects = msg.projects
		m.note = msg.note
		m.rebuildProj()
		return m, nil
	case scannedMsg:
		m.busy = ""
		m.projects = msg.projects
		m.rebuildProj()
		if m.pendingOpen >= 0 {
			i := m.pendingOpen
			m.pendingOpen = -1
			if msg.failed != "" && (i >= len(m.projects) || !m.projects[i].Scanned) {
				m.status = "Could not read " + msg.failed
				return m, nil
			}
			if i >= 0 && i < len(m.projects) && m.projects[i].Scanned {
				m.enterItems(i)
			}
			return m, nil
		}
		if m.pendingWipe {
			m.pendingWipe = false
			if msg.failed != "" {
				m.status = "Could not read " + msg.failed
				return m, nil
			}
			return m.beginWipe()
		}
		if msg.failed != "" {
			m.status = "Could not read " + msg.failed
		} else {
			m.status = "Scanned"
		}
		return m, nil
	case delMsg:
		if msg.done {
			m.finishDelete()
			return m, nil
		}
		if msg.ok {
			m.markDeleted(m.jobs[m.jobi])
		} else if msg.err != "" {
			m.fails++
			m.failLines = append(m.failLines, msg.err)
		}
		m.jobi++
		if m.stop || m.jobi >= len(m.jobs) {
			m.finishDelete()
			return m, nil
		}
		return m, m.deleteCmd()
	case actionsListedMsg:
		return m.onActionsListed(msg)
	case actionAppliedMsg:
		return m.applyNext(msg.err)
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			if m.screen == scrAction && m.act.stage == stageApplying {
				m.act.stopped = true
				m.status = "Stopping after the current request…"
				return m, nil
			}
			if m.screen == scrWorking {
				m.stop = true
				m.status = "Stopping after the current request…"
				return m, nil
			}
			return m, tea.Quit
		}
		return m.onKey(msg)
	}
	if m.screen == scrPrompt || (m.screen == scrConfirm && m.confirm.strict) {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) onKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := k.String()
	switch m.screen {
	case scrHelp:
		if s == "q" {
			return m, tea.Quit
		}
		m.screen = m.helpFrom
		return m, nil
	case scrReport:
		return m.onReport(s)
	case scrAction:
		return m.onAction(s)
	case scrPreferences:
		return m.onPreferences(s)
	case scrResult:
		if s == "q" {
			return m, tea.Quit
		}
		if m.open >= 0 && m.confirm.scope == "items" {
			m.screen = scrItems
			m.rows = m.projects[m.open].Rows(m.filter, m.opts.WantArt, m.opts.WantCache)
			m.clampCursor()
		} else {
			m.screen = scrProjects
			m.rebuildProj()
		}
		return m, nil
	case scrWorking:
		return m, nil
	case scrPrompt:
		return m.onPrompt(s)
	case scrConfirm:
		return m.onConfirm(s)
	case scrItems:
		if m.filterEdit {
			return m.onFilter(&m.filter, &m.filterEdit, s, true)
		}
		return m.onItems(s)
	default:
		if m.projFilterEdit {
			return m.onFilter(&m.projFilter, &m.projFilterEdit, s, false)
		}
		return m.onProjects(s)
	}
}

func (m model) onFilter(buf *string, edit *bool, s string, items bool) (tea.Model, tea.Cmd) {
	m.status = ""
	switch s {
	case "enter":
		*edit = false
	case "esc":
		*buf = ""
		*edit = false
		m.afterFilter(items)
	case "backspace":
		r := []rune(*buf)
		if len(r) > 0 {
			*buf = string(r[:len(r)-1])
		}
		m.afterFilter(items)
	case "ctrl+u":
		*buf = ""
		m.afterFilter(items)
	case "up":
		m.move(-1)
	case "down":
		m.move(1)
	default:
		if len([]rune(s)) == 1 && s[0] >= 32 {
			*buf += s
			m.afterFilter(items)
		}
	}
	return m, nil
}

func (m *model) afterFilter(items bool) {
	if items {
		if m.open >= 0 {
			m.rows = m.projects[m.open].Rows(m.filter, m.opts.WantArt, m.opts.WantCache)
		}
	} else {
		m.rebuildProj()
	}
	m.clampCursor()
}

func (m model) onProjects(s string) (tea.Model, tea.Cmd) {
	m.status = ""
	switch s {
	case "q":
		return m, tea.Quit
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "g", "home":
		m.cursor = 0
		m.ensure(m.rowCount())
	case "G", "end":
		if n := m.rowCount(); n > 0 {
			m.cursor = n - 1
		}
		m.ensure(m.rowCount())
	case " ":
		m.toggleProject()
	case "a":
		m.setVisibleProjects(true)
	case "n", "N":
		m.setVisibleProjects(false)
	case "enter", "right", "l":
		return m.openCurrent()
	case "d":
		return m.startWipe()
	case "s":
		return m.scanMarkedOrCursor()
	case "+", "=":
		return m.startPrompt("add", "Add a repository", "owner/repo")
	case "/":
		m.projFilterEdit = true
	case "esc":
		if m.projFilter != "" {
			m.projFilter = ""
			m.projFilterEdit = false
			m.rebuildProj()
		}
	case "u":
		m.openUsage()
	case "m":
		return m.openActions()
	case "p":
		return m.openPreferences()
	case "?":
		m.helpFrom = scrProjects
		m.screen = scrHelp
	}
	return m, nil
}

func (m model) onItems(s string) (tea.Model, tea.Cmd) {
	m.status = ""
	switch s {
	case "q":
		return m, tea.Quit
	case "b", "left", "h":
		if s == "h" || s == "left" {
			m.collapseCurrent()
			return m, nil
		}
		m.screen = scrProjects
		m.rebuildProj()
		return m, nil
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "g", "home":
		m.cursor = 0
		m.ensure(m.rowCount())
	case "G", "end":
		if n := m.rowCount(); n > 0 {
			m.cursor = n - 1
		}
		m.ensure(m.rowCount())
	case " ":
		m.toggleRow()
	case "enter":
		m.enterRow()
	case "right", "l":
		m.expandCurrent()
	case "e":
		m.setOpen(true)
	case "c":
		m.setOpen(false)
	case "a":
		m.selectVisible("")
	case "A":
		m.selectVisible("artifact")
	case "C":
		m.selectVisible("cache")
	case "n":
		m.selectVisibleClear()
	case "N":
		m.clearAll()
	case "x":
		n := m.projects[m.open].SelectExpired(m.filter)
		m.status = fmt.Sprintf("Selected %d expired artifacts", n)
	case "o":
		return m.startPrompt("older", "Select artifacts older than how many days?", strconv.Itoa(project.Prefs().OldArtifactDays))
	case "t":
		return m.startPrompt("keep", "Keep how many of the newest of each artifact name?", strconv.Itoa(project.Prefs().KeepNewest))
	case "d":
		return m.startItemDelete()
	case "/":
		m.filterEdit = true
	case "r":
		m.busy = "Refreshing " + m.projects[m.open].NWO + "…"
		m.pendingOpen = m.open
		return m, m.scanCmd([]int{m.open})
	case "esc":
		if m.filter != "" {
			m.filter = ""
			m.filterEdit = false
			m.rows = m.projects[m.open].Rows(m.filter, m.opts.WantArt, m.opts.WantCache)
			m.clampCursor()
		}
	case "i":
		m.openArtifactList()
	case "?":
		m.helpFrom = scrItems
		m.screen = scrHelp
	}
	return m, nil
}

func (m model) onPrompt(s string) (tea.Model, tea.Cmd) {
	if s == "esc" || (s == "q" && m.promptKind != "add") {
		m.screen = m.promptFrom
		return m, nil
	}
	switch s {
	case "enter":
		v := strings.TrimSpace(m.input.Value())
		switch m.promptKind {
		case "add":
			p, err := project.Parse(v)
			if err != nil {
				m.status = err.Error()
				m.screen = scrProjects
				return m, nil
			}
			for _, existing := range m.projects {
				if existing.NWO == p.NWO {
					m.status = "Already listed"
					m.screen = scrProjects
					return m, nil
				}
			}
			m.projects = append(m.projects, p)
			m.rebuildProj()
			m.status = "Added " + p.NWO
			m.screen = scrProjects
		case "older", "keep":
			if !allDigits(v) {
				m.status = "Not a number of days"
				m.screen = scrItems
				return m, nil
			}
			days, _ := strconv.Atoi(v)
			p := &m.projects[m.open]
			if m.promptKind == "keep" {
				n := p.SelectExceptNewest(days, m.filter)
				m.status = fmt.Sprintf("Selected %d artifacts, keeping the newest %d of each name", n, days)
			} else {
				n := p.SelectOlder(days, m.filter)
				m.status = fmt.Sprintf("Selected %d artifacts older than %d days", n, days)
			}
			m.screen = scrItems
		}
		return m, nil
	default:
		if m.promptKind != "add" {
			if len(s) == 1 && (s[0] < '0' || s[0] > '9') && s != "backspace" {
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(tea.KeyMsg(keyFromString(s)))
		return m, cmd
	}
}

func (m model) onConfirm(s string) (tea.Model, tea.Cmd) {
	m.status = ""
	if m.confirm.strict {
		switch s {
		case "enter":
			v := strings.TrimSpace(m.input.Value())
			if v == m.confirm.token || (m.confirm.scope == "items" && v == m.confirm.nwo) {
				return m.beginDelete()
			}
			m.status = "Type " + m.confirm.token + " to confirm"
			return m, nil
		case "esc", "q":
			m.screen = m.returnScreen()
			return m, nil
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(tea.KeyMsg(keyFromString(s)))
			return m, cmd
		}
	}
	switch s {
	case "y", "Y":
		return m.beginDelete()
	case "n", "N", "esc", "q", "enter":
		m.screen = m.returnScreen()
	}
	return m, nil
}

func keyFromString(s string) tea.Key {
	switch s {
	case "enter":
		return tea.Key{Type: tea.KeyEnter}
	case "esc":
		return tea.Key{Type: tea.KeyEsc}
	case "backspace":
		return tea.Key{Type: tea.KeyBackspace}
	case " ":
		return tea.Key{Type: tea.KeySpace, Runes: []rune{' '}}
	default:
		return tea.Key{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func (m model) returnScreen() screen {
	if m.confirm.scope == "items" {
		return scrItems
	}
	return scrProjects
}

func (m *model) rebuildProj() {
	m.projView = nil
	f := strings.ToLower(strings.TrimSpace(m.projFilter))
	for i, p := range m.projects {
		if f == "" || strings.Contains(strings.ToLower(p.NWO), f) {
			m.projView = append(m.projView, i)
		}
	}
	if m.screen == scrProjects {
		m.clampCursor()
	}
}

func (m *model) rowCount() int {
	if m.screen == scrItems {
		return len(m.rows)
	}
	if m.screen == scrProjects {
		return len(m.projView)
	}
	return 0
}

func (m *model) listH() int {
	h := m.height - 7
	if h < 1 {
		return 1
	}
	return h
}

func (m *model) clampCursor() {
	n := m.rowCount()
	if n == 0 {
		m.cursor = 0
		m.scroll = 0
		return
	}
	if m.cursor >= n {
		m.cursor = n - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.ensure(n)
}

func (m *model) ensure(n int) {
	if n <= 0 {
		m.scroll = 0
		return
	}
	win := m.listH()
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+win {
		m.scroll = m.cursor - win + 1
	}
	max := n - win
	if max < 0 {
		max = 0
	}
	if m.scroll > max {
		m.scroll = max
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
}

func (m *model) move(d int) {
	n := m.rowCount()
	if n == 0 {
		return
	}
	m.cursor += d
	if m.cursor < 0 {
		m.cursor = n - 1
	} else if m.cursor >= n {
		m.cursor = 0
	}
	m.ensure(n)
}

func (m *model) toggleProject() {
	if len(m.projView) == 0 {
		return
	}
	i := m.projView[m.cursor]
	m.projects[i].On = !m.projects[i].On
}

func (m *model) setVisibleProjects(on bool) {
	for _, i := range m.projView {
		m.projects[i].On = on
	}
}

func (m model) scanCmd(idxs []int) tea.Cmd {
	projects := append([]project.Project(nil), m.projects...)
	wantArt, wantCache := m.opts.WantArt, m.opts.WantCache
	return func() tea.Msg {
		failed := ""
		for _, i := range idxs {
			p, err := gh.Load(projects[i], wantArt, wantCache)
			if err != nil {
				projects[i].Err = err.Error()
				projects[i].Scanned = false
				if failed == "" {
					failed = projects[i].NWO
				}
				continue
			}
			projects[i] = p
		}
		return scannedMsg{projects: projects, failed: failed}
	}
}

func (m model) openCurrent() (model, tea.Cmd) {
	if len(m.projView) == 0 {
		return m, nil
	}
	i := m.projView[m.cursor]
	if !m.projects[i].Scanned {
		m.busy = "Reading " + m.projects[i].NWO + "…"
		m.pendingOpen = i
		return m, m.scanCmd([]int{i})
	}
	m.enterItems(i)
	return m, nil
}

func (m *model) enterItems(i int) {
	m.open = i
	m.screen = scrItems
	m.rows = m.projects[i].Rows(m.filter, m.opts.WantArt, m.opts.WantCache)
	m.cursor = 0
	m.scroll = 0
	for n, r := range m.rows {
		if r.Kind == project.RowGroup || r.Kind == project.RowItem {
			m.cursor = n
			break
		}
	}
	m.ensure(len(m.rows))
}

func (m model) scanMarkedOrCursor() (model, tea.Cmd) {
	var idxs []int
	for i, p := range m.projects {
		if p.On {
			idxs = append(idxs, i)
		}
	}
	if len(idxs) == 0 && len(m.projView) > 0 {
		idxs = []int{m.projView[m.cursor]}
	}
	if len(idxs) == 0 {
		m.status = "Nothing to scan"
		return m, nil
	}
	m.busy = "Reading…"
	return m, m.scanCmd(idxs)
}

func (m model) startWipe() (model, tea.Cmd) {
	var idxs []int
	marked := 0
	for i, p := range m.projects {
		if !p.On {
			continue
		}
		marked++
		if !p.Scanned {
			idxs = append(idxs, i)
		}
	}
	if marked == 0 {
		m.status = "Nothing marked"
		return m, nil
	}
	if len(idxs) > 0 {
		m.pendingWipe = true
		m.busy = "Reading marked projects…"
		return m, m.scanCmd(idxs)
	}
	return m.beginWipe()
}

func (m model) beginWipe() (model, tea.Cmd) {
	var lines []string
	var n, items int
	var bytes, expired int64
	var only project.Project
	for _, p := range m.projects {
		if !p.On {
			continue
		}
		n++
		only = p
		items += p.ArtN + p.CacheN
		bytes += p.ArtBytes + p.CacheBytes
		expired += p.ExpBytes
		lines = append(lines, fmt.Sprintf("  %s    %d artifacts    %d caches    %s",
			p.NWO, p.ArtN, p.CacheN, format.HumanSize(p.ArtBytes+p.CacheBytes)))
	}
	if items == 0 {
		m.status = "Those projects have no artifacts or caches"
		m.screen = scrProjects
		return m, nil
	}
	m.confirm = confirmState{
		scope: "projects", strict: n > 1 || items > project.Prefs().ConfirmEntries || bytes >= project.Prefs().ConfirmBytes,
		count: items, bytes: bytes, expired: expired, lines: lines, nwo: only.NWO,
	}
	if n > 1 {
		m.confirm.strict = true
		m.confirm.token = "wipe"
	} else if m.confirm.strict {
		m.confirm.token = only.Name
	}
	m.screen = scrConfirm
	if m.confirm.strict {
		m.input = newInput(m.confirm.token)
		return m, textinput.Blink
	}
	return m, nil
}

func (m model) startItemDelete() (model, tea.Cmd) {
	p := &m.projects[m.open]
	n, bytes, _, _ := p.Selected()
	if n == 0 {
		m.status = "Nothing selected"
		return m, nil
	}
	var lines []string
	for _, gi := range append(append([]int{}, p.ArtGroups...), p.CacheGroups...) {
		g := p.Groups[gi]
		on := 0
		var sz int64
		for i := g.Start; i < g.Start+g.Len; i++ {
			it := p.Items[i]
			if it.Dead || !it.On {
				continue
			}
			on++
			sz += it.Size
		}
		if on == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("  %s    %d    %s", g.Name, on, format.HumanSize(sz)))
	}
	m.confirm = confirmState{
		scope: "items", strict: n > project.Prefs().ConfirmEntries || bytes >= project.Prefs().ConfirmBytes,
		token: p.Name, nwo: p.NWO, count: n, bytes: bytes, lines: lines,
	}
	m.screen = scrConfirm
	if m.confirm.strict {
		m.input = newInput(p.Name)
		return m, textinput.Blink
	}
	return m, nil
}

func newInput(placeholder string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 200
	ti.Width = 40
	ti.Focus()
	return ti
}

func (m model) startPrompt(kind, title, placeholder string) (model, tea.Cmd) {
	m.promptKind = kind
	m.promptFrom = m.screen
	m.screen = scrPrompt
	m.input = newInput(placeholder)
	if kind != "add" {
		m.input.SetValue(placeholder)
	}
	m.status = title
	return m, textinput.Blink
}

func (m model) beginDelete() (model, tea.Cmd) {
	m.jobs = m.buildJobs()
	if len(m.jobs) == 0 {
		m.status = "Nothing selected"
		m.screen = m.returnScreen()
		return m, nil
	}
	sortJobs(m.jobs)
	m.jobi = 0
	m.freed = 0
	m.fails = 0
	m.failLines = nil
	m.okArt = 0
	m.okCache = 0
	m.stop = false
	m.screen = scrWorking
	return m, m.deleteCmd()
}

func sortJobs(jobs []delJob) {
	// Largest first. Equal sizes keep the existing order.
	for i := 1; i < len(jobs); i++ {
		j := i
		for j > 0 && jobs[j].size > jobs[j-1].size {
			jobs[j], jobs[j-1] = jobs[j-1], jobs[j]
			j--
		}
	}
}

func (m model) buildJobs() []delJob {
	var jobs []delJob
	add := func(pi int, p project.Project, it project.Item) {
		if it.Dead {
			return
		}
		if m.confirm.scope == "items" && !it.On {
			return
		}
		jobs = append(jobs, delJob{
			proj: pi, id: it.ID, kind: it.Kind, name: it.Name, nwo: p.NWO,
			size: it.Size, apiHost: p.APIHost, owner: p.Owner, repoName: p.Name,
		})
	}
	if m.confirm.scope == "projects" {
		for pi, p := range m.projects {
			if !p.On {
				continue
			}
			for _, it := range p.Items {
				add(pi, p, it)
			}
		}
		return jobs
	}
	if m.open < 0 {
		return nil
	}
	p := m.projects[m.open]
	for _, it := range p.Items {
		add(m.open, p, it)
	}
	return jobs
}

func (m model) deleteCmd() tea.Cmd {
	if m.jobi >= len(m.jobs) {
		return func() tea.Msg { return delMsg{done: true} }
	}
	job := m.jobs[m.jobi]
	dry := m.opts.DryRun
	return func() tea.Msg {
		if m.stop {
			return delMsg{done: true}
		}
		label := job.nwo + "  " + job.name
		var err error
		if !dry {
			p := project.Project{Owner: job.owner, Name: job.repoName, APIHost: job.apiHost, NWO: job.nwo}
			if job.kind == "artifact" {
				err = gh.DeleteArtifact(p, job.id)
			} else {
				err = gh.DeleteCache(p, job.id)
			}
		}
		if err != nil {
			return delMsg{ok: false, err: fmt.Sprintf("%s #%s  %s  %s", job.kind, job.id, format.Fit(job.name, 24), format.Fit(err.Error(), 48)), label: label}
		}
		return delMsg{ok: true, label: label}
	}
}

func (m *model) markDeleted(job delJob) {
	p := &m.projects[job.proj]
	for i := range p.Items {
		if p.Items[i].Kind == job.kind && p.Items[i].ID == job.id {
			p.Items[i].Dead = true
			p.Items[i].On = false
		}
	}
	p.Recompute()
	m.freed += job.size
	if job.kind == "artifact" {
		m.okArt++
		m.sessionArt++
	} else {
		m.okCache++
		m.sessionCache++
	}
	m.sessionBytes += job.size
	if !strings.Contains(m.sessionRepos, job.nwo) {
		if m.sessionRepos != "" {
			m.sessionRepos += ", "
		}
		m.sessionRepos += job.nwo
	}
}

func (m *model) finishDelete() {
	marked := countMarked(m.projects)
	for i := range m.projects {
		if m.projects[i].Scanned && m.projects[i].ArtN+m.projects[i].CacheN == 0 {
			m.projects[i].On = false
		}
	}
	if m.confirm.scope == "projects" && marked > 1 {
		m.resultWhere = fmt.Sprintf("%d projects", marked)
	} else if m.confirm.nwo != "" {
		m.resultWhere = m.confirm.nwo
	}
	if m.open >= 0 && m.open < len(m.projects) {
		m.projects[m.open].RebuildGroups()
		m.rows = m.projects[m.open].Rows(m.filter, m.opts.WantArt, m.opts.WantCache)
	}
	m.screen = scrResult
}

func countMarked(ps []project.Project) int {
	n := 0
	for _, p := range ps {
		if p.On {
			n++
		}
	}
	if n == 0 {
		return 1
	}
	return n
}

func (m *model) toggleRow() {
	if m.open < 0 || m.cursor < 0 || m.cursor >= len(m.rows) {
		return
	}
	p := &m.projects[m.open]
	r := m.rows[m.cursor]
	switch r.Kind {
	case project.RowSection:
		kind := "artifact"
		if r.Ref == 1 {
			kind = "cache"
		}
		p.ToggleMatching(func(it project.Item) bool { return it.Kind == kind && it.Matches(m.filter) })
	case project.RowGroup:
		gi := r.Ref
		g := p.Groups[gi]
		p.ToggleMatching(func(it project.Item) bool {
			return it.Group == gi && it.Matches(m.filter) && it.Name == g.Name
		})
	case project.RowItem:
		p.ToggleItem(r.Ref)
	}
}

func (m *model) enterRow() {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return
	}
	r := m.rows[m.cursor]
	if r.Kind != project.RowGroup {
		m.toggleRow()
		return
	}
	m.projects[m.open].Groups[r.Ref].Open = !m.projects[m.open].Groups[r.Ref].Open
	m.rows = m.projects[m.open].Rows(m.filter, m.opts.WantArt, m.opts.WantCache)
	m.clampCursor()
}

func (m *model) expandCurrent() {
	m.setCurrentOpen(true)
}

func (m *model) collapseCurrent() {
	m.setCurrentOpen(false)
}

func (m *model) setCurrentOpen(open bool) {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return
	}
	r := m.rows[m.cursor]
	gi := -1
	switch r.Kind {
	case project.RowGroup:
		gi = r.Ref
	case project.RowItem:
		gi = m.projects[m.open].Items[r.Ref].Group
	}
	if gi < 0 {
		return
	}
	m.projects[m.open].Groups[gi].Open = open
	m.rows = m.projects[m.open].Rows(m.filter, m.opts.WantArt, m.opts.WantCache)
	if !open {
		for i, row := range m.rows {
			if row.Kind == project.RowGroup && row.Ref == gi {
				m.cursor = i
				break
			}
		}
	}
	m.clampCursor()
}

func (m *model) setOpen(open bool) {
	if m.open < 0 {
		return
	}
	for i := range m.projects[m.open].Groups {
		m.projects[m.open].Groups[i].Open = open
	}
	m.rows = m.projects[m.open].Rows(m.filter, m.opts.WantArt, m.opts.WantCache)
	m.clampCursor()
}

func (m *model) selectVisible(kind string) {
	p := &m.projects[m.open]
	for i := range p.Items {
		it := &p.Items[i]
		if it.Dead || !it.Matches(m.filter) {
			continue
		}
		if kind != "" && it.Kind != kind {
			continue
		}
		it.On = true
	}
}

func (m *model) selectVisibleClear() {
	p := &m.projects[m.open]
	for i := range p.Items {
		if p.Items[i].Matches(m.filter) {
			p.Items[i].On = false
		}
	}
}

func (m *model) clearAll() {
	p := &m.projects[m.open]
	for i := range p.Items {
		p.Items[i].On = false
	}
}

func (m model) View() string {
	if !m.ready {
		return titleStyle.Render("gh-sweep") + "\n\n  Loading projects…\n"
	}
	if m.loadErr != "" {
		return titleStyle.Render("gh-sweep") + "\n\n" + errStyle.Render(m.loadErr) + "\n\n" + dimStyle.Render("q quit") + "\n"
	}
	switch m.screen {
	case scrItems:
		return m.viewItems()
	case scrConfirm:
		return m.viewConfirm()
	case scrWorking:
		return m.viewWorking()
	case scrResult:
		return m.viewResult()
	case scrHelp:
		return m.viewHelp()
	case scrPrompt:
		return m.viewPrompt()
	case scrReport:
		return m.viewReport()
	case scrAction:
		return m.viewAction()
	case scrPreferences:
		return m.viewPreferences()
	default:
		return m.viewProjects()
	}
}

func (m model) viewProjects() string {
	var b strings.Builder
	head := "gh-sweep  projects"
	if m.opts.DryRun {
		head += "   dry-run"
	}
	b.WriteString(titleStyle.Render("gh-sweep") + nameStyle.Render("  projects"))
	if m.opts.DryRun {
		b.WriteString(warnStyle.Render("   dry-run"))
	}
	b.WriteByte('\n')
	switch {
	case m.busy != "":
		b.WriteString(warnStyle.Render(m.busy))
	case m.projFilter != "" || m.projFilterEdit:
		line := "/ " + m.projFilter
		if m.projFilterEdit {
			line += "_"
		}
		line += fmt.Sprintf("   %d shown", len(m.projView))
		b.WriteString(line)
	case m.note != "":
		b.WriteString(warnStyle.Render(m.note))
	default:
		b.WriteString(dimStyle.Render("space wipes a whole project    enter picks artifacts and caches"))
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Expired artifacts still count against the Actions quota."))
	b.WriteString("\n\n")
	if len(m.projView) == 0 {
		b.WriteString("  No projects yet. Press + and type owner/repo.\n")
	} else {
		win := m.listH()
		for i := 0; i < win && m.scroll+i < len(m.projView); i++ {
			idx := m.scroll + i
			b.WriteString(m.projectLine(m.projView[idx], idx == m.cursor))
			b.WriteByte('\n')
		}
	}
	b.WriteByte('\n')
	sum := m.projectSummary()
	if m.status != "" {
		sum = m.status
	}
	b.WriteString(dimStyle.Render(format.Fit(sum, m.width-1)))
	b.WriteByte('\n')
	keys := "space mark    enter pick    a all    n none    d wipe    s scan    u usage    m actions    p preferences    + add    / find    q quit"
	if m.projFilterEdit {
		keys = "type to filter    enter done    esc clear"
	}
	b.WriteString(dimStyle.Render(format.Fit(keys, m.width-1)))
	b.WriteByte('\n')
	_ = head
	return b.String()
}

func (m model) projectLine(i int, cur bool) string {
	p := m.projects[i]
	box := "none"
	if p.On {
		box = "all"
	}
	right := "not scanned"
	if p.Err != "" && !p.Scanned {
		right = "unreadable"
	} else if p.Scanned {
		right = fmt.Sprintf("%d artifacts · %s", p.ArtN, format.HumanSize(p.ArtBytes))
		if m.opts.WantCache {
			right += fmt.Sprintf(" · %d caches · %s", p.CacheN, format.HumanSize(p.CacheBytes))
		}
		if p.ExpN > 0 {
			right += " · " + format.HumanSize(p.ExpBytes) + " expired"
		}
	}
	return m.renderRow(0, cur, box, p.NWO, right, true)
}

func (m model) projectSummary() string {
	n, known, unscanned := 0, 0, 0
	var bytes int64
	for _, p := range m.projects {
		if !p.On {
			continue
		}
		n++
		if p.Scanned {
			known++
			bytes += p.ArtBytes + p.CacheBytes
		} else {
			unscanned++
		}
	}
	if n == 0 {
		return "nothing marked"
	}
	s := fmt.Sprintf("%d projects marked", n)
	if known > 0 {
		s += " · " + format.HumanSize(bytes) + " scanned"
	}
	if unscanned > 0 {
		s += fmt.Sprintf(" · %d not scanned", unscanned)
	}
	return s
}

func (m model) viewItems() string {
	p := m.projects[m.open]
	var b strings.Builder
	b.WriteString(titleStyle.Render("gh-sweep") + "  " + nameStyle.Render(p.NWO))
	if m.opts.DryRun {
		b.WriteString(warnStyle.Render("   dry-run"))
	}
	b.WriteByte('\n')
	if m.busy != "" {
		b.WriteString(warnStyle.Render(m.busy) + "\n")
	} else if m.filter != "" || m.filterEdit {
		line := "/ " + m.filter
		if m.filterEdit {
			line += "_"
		}
		shown := 0
		for _, it := range p.Items {
			if !it.Dead && it.Matches(m.filter) {
				shown++
			}
		}
		line += fmt.Sprintf("   %d shown", shown)
		b.WriteString(line + "\n")
	} else if p.Err != "" {
		b.WriteString(warnStyle.Render(format.Fit(p.Err, m.width-1)) + "\n")
	} else {
		b.WriteString(dimStyle.Render(fmt.Sprintf("%d artifacts · %s · %d caches · %s",
			p.ArtN, format.HumanSize(p.ArtBytes), p.CacheN, format.HumanSize(p.CacheBytes))) + "\n")
	}
	b.WriteByte('\n')
	if len(m.rows) == 0 {
		if m.filter != "" {
			b.WriteString("  No matches.\n")
		} else {
			b.WriteString("  Nothing to clean in this repository.\n")
		}
	} else {
		win := m.listH()
		for i := 0; i < win && m.scroll+i < len(m.rows); i++ {
			idx := m.scroll + i
			b.WriteString(m.itemLine(idx == m.cursor, m.rows[idx]))
			b.WriteByte('\n')
		}
	}
	b.WriteByte('\n')
	n, bytes, arts, caches := p.Selected()
	sum := "nothing selected"
	if m.status != "" {
		sum = m.status
	} else if n > 0 {
		sum = fmt.Sprintf("%d selected · %s · %d artifacts · %d caches", n, format.HumanSize(bytes), arts, caches)
	}
	b.WriteString(dimStyle.Render(format.Fit(sum, m.width-1)) + "\n")
	keys := "space select    j/k move    enter open    d delete    t trim    x expired    i details    b back    q quit"
	if m.filterEdit {
		keys = "type to filter    enter done    esc clear    arrows move"
	}
	b.WriteString(dimStyle.Render(format.Fit(keys, m.width-1)) + "\n")
	return b.String()
}

func (m model) itemLine(cur bool, r project.ViewRow) string {
	p := m.projects[m.open]
	switch r.Kind {
	case project.RowSection:
		kind := "artifact"
		label := "ARTIFACTS"
		if r.Ref == 1 {
			kind = "cache"
			label = "CACHES"
		}
		live, on, bytes := p.StateOf(func(it project.Item) bool { return it.Kind == kind && it.Matches(m.filter) })
		noun := "items"
		if kind == "cache" {
			noun = "entries"
		}
		right := fmt.Sprintf("%d %s · %s", live, noun, format.HumanSize(bytes))
		if on > 0 {
			right = fmt.Sprintf("%d/%d %s · %s", on, live, noun, format.HumanSize(bytes))
		}
		return m.renderRow(0, cur, project.BoxState(live, on), label, right, true)
	case project.RowGroup:
		g := p.Groups[r.Ref]
		live, on, bytes := 0, 0, int64(0)
		for i := g.Start; i < g.Start+g.Len; i++ {
			it := p.Items[i]
			if it.Dead || !it.Matches(m.filter) {
				continue
			}
			live++
			bytes += it.Size
			if it.On {
				on++
			}
		}
		right := fmt.Sprintf("%d · %s", live, format.HumanSize(bytes))
		if on > 0 && on < live {
			right = fmt.Sprintf("%d/%d · %s", on, live, format.HumanSize(bytes))
		}
		mark := g.Name
		if g.Open {
			mark = "▾ " + g.Name
		} else {
			mark = "▸ " + g.Name
		}
		return m.renderRow(1, cur, project.BoxState(live, on), mark, right, false)
	default:
		it := p.Items[r.Ref]
		state := "none"
		if it.On {
			state = "all"
		}
		right := fmt.Sprintf("%s · %s · %s · #%s", it.Meta, format.HumanTime(it.When), format.HumanSize(it.Size), it.ID)
		if it.Large() {
			right += " · large"
		}
		if it.Expired {
			right += " · expired"
		} else if !it.Expires.IsZero() {
			right += " · expires " + format.Date(it.Expires)
		}
		return m.renderRow(2, cur, state, it.Name, right, false)
	}
}

func (m model) renderRow(level int, cur bool, state, label, right string, section bool) string {
	indent := strings.Repeat("  ", level)
	mark := " "
	if cur {
		mark = cursorStyle.Render(">")
	}
	box := styledBox(state)
	usable := m.width - 1
	if usable < 20 {
		usable = 20
	}
	prefix := indent + "  "
	// visible prefix width: indent + mark + space + checkbox + space
	chrome := len([]rune(indent)) + 1 + 1 + 3 + 1
	rw := len([]rune(right))
	lw := usable - chrome - 2 - rw
	if lw < 8 {
		lw = 8
		if chrome+2+lw > usable {
			right = ""
			rw = 0
			lw = usable - chrome
		} else {
			right = format.Fit(right, usable-chrome-2-lw)
			rw = len([]rune(right))
			lw = usable - chrome - 2 - rw
		}
	}
	if lw < 1 {
		lw = 1
	}
	lab := format.PadRight(format.Fit(label, lw), lw)
	if section {
		lab = sectionStyle.Render(lab)
	} else if cur {
		lab = nameStyle.Render(lab)
	}
	return prefix[:len(indent)] + mark + " " + box + " " + lab + "  " + styleMeta(right, cur)
}

// warnTags are the parts of a row's right side that stand in the warning
// color: an expired artifact, and a large artifact.
var warnTags = map[string]bool{"expired": true, "large": true}

// styleMeta draws the parts of the right side of a row. A part in warnTags
// is in the warning color, and the other parts are faint. With no such part,
// the row under the cursor keeps its right side plain.
func styleMeta(right string, cur bool) string {
	parts := strings.Split(right, " · ")
	warn := false
	for _, part := range parts {
		warn = warn || warnTags[strings.TrimSpace(part)]
	}
	if !warn {
		if cur {
			return right
		}
		return dimStyle.Render(right)
	}
	styled := make([]string, len(parts))
	for i, part := range parts {
		if warnTags[strings.TrimSpace(part)] {
			styled[i] = warnStyle.Render(part)
		} else {
			styled[i] = dimStyle.Render(part)
		}
	}
	return strings.Join(styled, dimStyle.Render(" · "))
}

func (m model) viewConfirm() string {
	var b strings.Builder
	if m.confirm.token == "wipe" {
		b.WriteString(errStyle.Render("Wipe marked projects") + "\n\n")
	} else {
		b.WriteString(errStyle.Render("Delete from "+m.confirm.nwo) + "\n\n")
	}
	if m.opts.DryRun {
		b.WriteString(warnStyle.Render("Dry run: nothing will be deleted.") + "\n")
	} else if m.confirm.scope == "projects" {
		b.WriteString("This permanently deletes every artifact and cache in the marked projects.\n")
	} else {
		b.WriteString("This permanently deletes the selection. GitHub cannot undo it.\n")
	}
	if m.confirm.expired > 0 {
		b.WriteString(warnStyle.Render(format.HumanSize(m.confirm.expired)+" is expired artifacts. They still use quota.") + "\n")
	}
	b.WriteByte('\n')
	maxShow := m.height - 14
	if maxShow < 3 {
		maxShow = 3
	}
	for i, line := range m.confirm.lines {
		if i >= maxShow {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  … %d more\n", len(m.confirm.lines)-i)))
			break
		}
		b.WriteString(format.Fit(line, m.width-1) + "\n")
	}
	b.WriteString(fmt.Sprintf("\nTotal  %d   %s\n", m.confirm.count, format.HumanSize(m.confirm.bytes)))
	b.WriteString(dimStyle.Render("Largest files are deleted first.") + "\n")
	if m.confirm.count > 200 {
		b.WriteString(dimStyle.Render("GitHub allows about 5,000 API calls an hour.") + "\n")
	}
	b.WriteByte('\n')
	if m.confirm.strict {
		b.WriteString("Type " + m.confirm.token + " to confirm\n")
		b.WriteString(m.input.View() + "\n")
		b.WriteString(dimStyle.Render("enter confirms    esc cancels") + "\n")
	} else if m.opts.DryRun {
		b.WriteString("y   finish the dry run\n")
		b.WriteString(dimStyle.Render("n   go back") + "\n")
	} else {
		b.WriteString(errStyle.Render("y   delete now") + "\n")
		b.WriteString(dimStyle.Render("n   go back") + "\n")
	}
	if m.status != "" {
		b.WriteString("\n" + warnStyle.Render(m.status) + "\n")
	}
	return b.String()
}

func (m model) viewWorking() string {
	total := len(m.jobs)
	cur := m.jobi
	if cur > total {
		cur = total
	}
	label := ""
	if m.jobi < len(m.jobs) {
		label = m.jobs[m.jobi].nwo + "  " + m.jobs[m.jobi].name
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Deleting %d of %d", cur, total)) + "\n\n")
	b.WriteString(label + "\n\n")
	width := 24
	filled := 0
	if total > 0 {
		filled = cur * width / total
	}
	bar := strings.Repeat("#", filled) + strings.Repeat("-", width-filled)
	b.WriteString("[" + bar + "]\n\n")
	b.WriteString(fmt.Sprintf("freed %s    failed %d\n", format.HumanSize(m.freed), m.fails))
	if m.status != "" {
		b.WriteString(warnStyle.Render(m.status) + "\n")
	}
	b.WriteString(dimStyle.Render("ctrl-c stops after the current request") + "\n")
	return b.String()
}

func (m model) viewResult() string {
	var b strings.Builder
	where := m.resultWhere
	if where == "" {
		where = "GitHub"
	}
	title := "Deleted from " + where
	style := okStyle
	if m.opts.DryRun {
		title = "Dry run for " + where
		style = warnStyle
	} else if m.stop {
		title = "Stopped early on " + where
		style = warnStyle
	} else if m.fails > 0 && m.okArt+m.okCache == 0 {
		title = "Nothing deleted on " + where
		style = errStyle
	}
	b.WriteString(style.Render(title) + "\n\n")
	if m.opts.DryRun {
		b.WriteString("No delete requests were sent.\n\n")
	}
	b.WriteString(fmt.Sprintf("  %d artifacts\n  %d caches\n  freed %s\n", m.okArt, m.okCache, format.HumanSize(m.freed)))
	if m.fails > 0 {
		b.WriteString("\n" + errStyle.Render("Failed") + "\n")
		for i, line := range m.failLines {
			if i >= 8 {
				b.WriteString(dimStyle.Render(fmt.Sprintf("  … %d more\n", len(m.failLines)-i)))
				break
			}
			b.WriteString("  " + line + "\n")
		}
	}
	b.WriteString("\n" + dimStyle.Render("Artifact quota usage refreshes every 6-12 hours.") + "\n")
	b.WriteString(dimStyle.Render("q quits    any other key goes back") + "\n")
	return b.String()
}

func (m model) viewHelp() string {
	text := `gh-sweep help

Projects: space marks a repo for a full wipe, enter opens it.
d on the project list deletes everything marked.
b returns from a project. + adds owner/repo.

space      toggle the highlighted row
enter      toggle an item, or open and close a group
j/k        move        h/l fold        e/c fold all
a/n        select or clear what the filter shows
x          select expired artifacts (they still use quota)
u          storage usage of the scanned projects
m          action menu: every write flag, report flag, and setting
p          preferences: limits, defaults, and confirmation sizes
i          details of each artifact in the open project
o          select artifacts older than N days
t          keep the newest N of each name, select the rest
d          delete       / filter       r refresh
?          help         q quit

A group checkbox selects every artifact of that name,
or every cache entry with that key.
Deletes run largest-first. Quota usage refreshes in 6-12 hours.

press a key to return     q quits`
	return text + "\n"
}

func (m model) viewPrompt() string {
	title := m.status
	if title == "" {
		title = "Answer"
	}
	hint := "Enter applies. Esc cancels."
	if m.promptKind == "add" {
		hint = "Type owner/repo. Enter adds it. Esc cancels."
	} else if m.promptKind == "keep" {
		hint = "0 selects every artifact of each name. Esc cancels."
	}
	return titleStyle.Render(title) + "\n\n" + m.input.View() + "\n\n" + dimStyle.Render(hint) + "\n"
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
