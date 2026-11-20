// id: D-4

package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rbe/gh-sweep/internal/format"
	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
)

func TestDeleteOrder(t *testing.T) {
	p := sampleProject()
	m := newModel(Options{WantArt: true, WantCache: true})
	m.projects = []project.Project{p}
	m.open = 0
	m.confirm.scope = "projects"
	m.projects[0].On = true
	jobs := m.buildJobs()
	sortJobs(jobs)
	var ids []string
	for _, j := range jobs {
		ids = append(ids, j.kind+":"+j.id)
	}
	want := []string{"cache:7", "artifact:101", "artifact:100", "cache:8", "cache:9", "artifact:90", "artifact:80"}
	if strings.Join(ids, ",") != strings.Join(want, ",") {
		t.Fatalf("order %v", ids)
	}
}

func TestUISelectsProjectAndGroup(t *testing.T) {
	p := sampleProject()
	p.On = false
	other, err := project.Parse("other/lib")
	if err != nil {
		t.Fatal(err)
	}
	m := newModel(Options{WantArt: true, WantCache: true})
	m.ready = true
	m.projects = []project.Project{p, other}
	m.rebuildProj()
	m.screen = scrProjects
	m.width, m.height = 100, 36
	next, _ := m.Update(teaKey(" "))
	m = next.(model)
	if !m.projects[0].On || m.projects[1].On {
		t.Fatal("space should mark only the current project")
	}
	if !strings.Contains(stripANSI(m.View()), "[x]") || !strings.Contains(stripANSI(m.View()), "acme/widgets") {
		t.Fatalf("view\n%s", stripANSI(m.View()))
	}
	m.enterItems(0)
	next, _ = m.Update(teaKey(" "))
	m = next.(model)
	if !itemOn(m.projects[0], "101") || !itemOn(m.projects[0], "100") || itemOn(m.projects[0], "90") {
		t.Fatal("space on the dist group should select both runs only")
	}
}

func sampleProject() project.Project {
	format.Now = func() time.Time { return time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC) }
	raw := strings.Join([]string{
		"T 4",
		strings.Join([]string{"101", "dist", "900000000", "2026-11-12T12:00:00Z", "0", "main", "500"}, "\x1f"),
		strings.Join([]string{"100", "dist", "600000000", "2026-11-10T12:00:00Z", "0", "release", "499"}, "\x1f"),
		strings.Join([]string{"90", "coverage", "12000000", "2026-10-01T12:00:00Z", "1", "", "0"}, "\x1f"),
		strings.Join([]string{"80", "Weird Name", "2048", "2026-11-14T11:00:00Z", "0", "feat/tabs", "1"}, "\x1f"),
	}, "\n")
	arts, total := gh.ParseArtifacts(raw)
	if total != 4 || len(arts) != 4 {
		panic("parse artifacts")
	}
	caches := gh.ParseCaches(strings.Join([]string{
		strings.Join([]string{"7", "node-Linux-abc", "2000000000", "2026-11-01T00:00:00Z", "refs/heads/main", "2026-11-13T12:00:00Z"}, "\x1f"),
		strings.Join([]string{"8", "node-Linux-abc", "100000000", "2026-10-01T00:00:00Z", "refs/pull/12/merge", "2026-10-02T00:00:00Z"}, "\x1f"),
		strings.Join([]string{"9", "pip-Linux", "50000000", "2026-09-01T00:00:00Z", "refs/heads/main", "2026-09-02T00:00:00Z"}, "\x1f"),
	}, "\n"))
	p := project.Project{Owner: "acme", Name: "widgets", NWO: "acme/widgets", On: true}
	p.Items = append(arts, caches...)
	p.RebuildGroups()
	return p
}

func itemOn(p project.Project, id string) bool {
	for _, it := range p.Items {
		if it.ID == id {
			return it.On
		}
	}
	return false
}

func teaKey(s string) tea.Msg {
	return tea.KeyMsg(keyFromString(s))
}

func stripANSI(s string) string {
	var b strings.Builder
	esc := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			esc = true
		case esc && (r == 'm' || r == 'h' || r == 'l'):
			esc = false
		case esc:
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
