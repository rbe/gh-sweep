// id: ADR-0015

package ui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rbe/gh-sweep/internal/project"
	"github.com/rbe/gh-sweep/internal/sweep"
)

// fakeRuns replaces the read of GitHub with two runs in each repository.
// The write of run 2 fails when failRun2 is set. writes records each write.
func fakeRuns(t *testing.T, failRun2 bool) *[]string {
	t.Helper()
	var writes []string
	var filters []project.Filter
	old := listEntries
	listEntries = func(a sweep.Action, p project.Project, f project.Filter) ([]sweep.Entry, error) {
		filters = append(filters, f)
		var entries []sweep.Entry
		for _, id := range []string{"1", "2"} {
			id, nwo := id, p.NWO
			entries = append(entries, sweep.Entry{
				Line: func(dry bool) string {
					if dry {
						return nwo + ": would delete run " + id
					}
					return nwo + ": deleted run " + id
				},
				Total: project.Total{Count: 1},
				Apply: func() error {
					if failRun2 && id == "2" {
						return errors.New("HTTP 403")
					}
					writes = append(writes, nwo+" "+id)
					return nil
				},
			})
		}
		return entries, nil
	}
	t.Cleanup(func() { listEntries = old })
	return &writes
}

func actionModel(o Options, names ...string) model {
	m := newModel(o)
	m.ready = true
	for _, name := range names {
		p, err := project.Parse(name)
		if err != nil {
			panic(err)
		}
		m.projects = append(m.projects, p)
	}
	m.rebuildProj()
	m.width, m.height = 160, 40
	return m
}

// press sends the keys and runs each command until no message follows.
func press(t *testing.T, m model, keys ...string) model {
	t.Helper()
	for _, k := range keys {
		next, cmd := m.Update(teaKey(k))
		m = next.(model)
		if m.screen == scrAction && (m.act.stage == stageListing || m.act.stage == stageApplying) {
			m = run(m, cmd)
		}
	}
	return m
}

func run(m model, cmd tea.Cmd) model {
	for cmd != nil {
		msg := cmd()
		switch msg.(type) {
		case actionsListedMsg, actionAppliedMsg:
		default:
			return m
		}
		next, c := m.Update(msg)
		m, cmd = next.(model), c
	}
	return m
}

func TestTheActionMenuDeletesOnlyTheCheckedWorkflowRuns(t *testing.T) {
	writes := fakeRuns(t, false)
	m := actionModel(Options{}, "acme/widgets")
	m = press(t, m, "m")
	if view := stripANSI(m.View()); !strings.Contains(view, "Delete workflow runs and their logs  --delete-runs") ||
		!strings.Contains(view, "Delete the files of releases  --delete-release-assets") {
		t.Fatalf("the menu does not list the actions\n%s", view)
	}
	m = press(t, m, "enter", "enter")
	if !strings.Contains(m.status, "--delete-runs needs at least one of --older-than, --branch, --status") {
		t.Fatalf("an empty form gives the status %q", m.status)
	}
	m = press(t, m, "3", "0", "enter")
	if m.act.stage != stagePreview || m.act.filter.OlderThan != 30 {
		t.Fatalf("stage %d, filter %+v", m.act.stage, m.act.filter)
	}
	view := stripANSI(m.View())
	for _, want := range []string{"[x] acme/widgets: would delete run 1", "[x] acme/widgets: would delete run 2", "checked 2 of 2"} {
		if !strings.Contains(view, want) {
			t.Fatalf("no %q in\n%s", want, view)
		}
	}
	m = press(t, m, "j", " ", "d")
	if m.act.stage != stageConfirm || m.act.token != "" {
		t.Fatalf("stage %d, token %q", m.act.stage, m.act.token)
	}
	m = press(t, m, "y")
	if strings.Join(*writes, ",") != "acme/widgets 1" {
		t.Fatalf("writes %v", *writes)
	}
	if m.screen != scrReport {
		t.Fatalf("screen %d after the writes", m.screen)
	}
	a, _ := sweep.ByFlag("--delete-runs")
	lines := strings.Join(m.report.lines, "\n")
	if lines != "acme/widgets: deleted run 1\n"+a.Total(project.Total{Count: 1}, false) {
		t.Fatalf("result lines\n%s", lines)
	}
	if next, _ := m.Update(teaKey("b")); next.(model).screen != scrProjects {
		t.Fatalf("b goes to screen %d", next.(model).screen)
	}
}

func TestTheActionMenuWithDryRunSendsNoWrite(t *testing.T) {
	writes := fakeRuns(t, false)
	m := actionModel(Options{DryRun: true}, "acme/widgets")
	m = press(t, m, "m", "enter", "3", "0", "enter", "d", "y")
	if len(*writes) != 0 {
		t.Fatalf("a dry run wrote %v", *writes)
	}
	if m.report.note != "Dry run: no request changed GitHub." || m.report.lines[0] != "acme/widgets: would delete run 1" {
		t.Fatalf("note %q, lines %v", m.report.note, m.report.lines)
	}
}

func TestTheActionMenuInTwoRepositoriesNeedsTheTypedVerb(t *testing.T) {
	writes := fakeRuns(t, false)
	m := actionModel(Options{}, "acme/widgets", "acme/lib")
	m.projects[0].On = true
	m.projects[1].On = true
	m = press(t, m, "m", "enter", "3", "0", "enter", "d", "y", "enter")
	if len(*writes) != 0 || m.act.token != "delete" || m.status != "Type delete to confirm" {
		t.Fatalf("token %q, status %q, writes %v", m.act.token, m.status, *writes)
	}
	m = press(t, m, "esc", "d", "d", "e", "l", "e", "t", "e", "enter")
	if len(*writes) != 4 {
		t.Fatalf("writes %v", *writes)
	}
}

func TestTheActionMenuListsAFailedWriteAndGoesOn(t *testing.T) {
	fakeRuns(t, true)
	m := actionModel(Options{}, "acme/widgets")
	m = press(t, m, "m", "enter", "3", "0", "enter", "d", "y")
	lines := strings.Join(m.report.lines, "\n")
	if !strings.Contains(lines, "acme/widgets: deleted run 1") || !strings.Contains(lines, "failed: acme/widgets: HTTP 403") {
		t.Fatalf("result lines\n%s", lines)
	}
}

func TestTheActionFormRefusesTheFilterValuesThatTheCommandLineRefuses(t *testing.T) {
	writes := fakeRuns(t, false)
	m := actionModel(Options{}, "acme/widgets")
	m = press(t, m, "m", "j", "j", "enter", "tab", "-", "1", "enter")
	if m.act.action.Flag != "--delete-releases" || m.act.stage != stageForm {
		t.Fatalf("action %s, stage %d", m.act.action.Flag, m.act.stage)
	}
	if m.status != "keep the newest: give a number of 0 or more" {
		t.Fatalf("status %q", m.status)
	}
	if len(*writes) != 0 {
		t.Fatalf("writes %v", *writes)
	}
}
