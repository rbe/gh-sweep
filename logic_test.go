package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestParseRepo(t *testing.T) {
	p, err := parseRepo("acme/widgets")
	if err != nil {
		t.Fatal(err)
	}
	if p.Owner != "acme" || p.Name != "widgets" || p.NWO != "acme/widgets" || p.APIHost != "" {
		t.Fatalf("%+v", p)
	}
	p, err = parseRepo("github.com/acme/widgets")
	if err != nil || p.NWO != "acme/widgets" || p.APIHost != "" {
		t.Fatalf("github.com: %+v %v", p, err)
	}
	p, err = parseRepo("ghe.example.com/acme/widgets.git")
	if err != nil || p.APIHost != "ghe.example.com" || p.NWO != "ghe.example.com/acme/widgets" {
		t.Fatalf("enterprise: %+v %v", p, err)
	}
	if _, err := parseRepo("not a repo"); err == nil {
		t.Fatal("expected error")
	}
}

func TestHumanSizeAndTime(t *testing.T) {
	if humanSize(0) != "0 B" || humanSize(1536) != "1.5 KB" || humanSize(1<<30) != "1.0 GB" {
		t.Fatalf("%q %q %q", humanSize(0), humanSize(1536), humanSize(1<<30))
	}
	nowFn = func() time.Time { return time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { nowFn = time.Now })
	got := humanTime(time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC))
	if got != "2d ago" {
		t.Fatalf("age %q", got)
	}
	when := parseGitHubTime("2020-01-10T14:59:22.0766667Z")
	if when.UTC().Format(time.RFC3339) != "2020-01-10T14:59:22Z" {
		t.Fatalf("fraction %v", when)
	}
	off := parseGitHubTime("2020-01-10T14:59:22+02:00")
	if !off.Equal(time.Date(2020, 1, 10, 12, 59, 22, 0, time.UTC)) {
		t.Fatalf("offset %v", off.UTC())
	}
	if shortenRef("refs/pull/12/merge") != "pr/12" || shortenRef("refs/heads/main") != "main" {
		t.Fatal(shortenRef("refs/pull/12/merge"), shortenRef("refs/heads/main"))
	}
}

func sampleProject() Project {
	nowFn = func() time.Time { return time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC) }
	raw := strings.Join([]string{
		"T 4",
		strings.Join([]string{"101", "dist", "900000000", "2026-11-12T12:00:00Z", "0", "main", "500"}, "\x1f"),
		strings.Join([]string{"100", "dist", "600000000", "2026-11-10T12:00:00Z", "0", "release", "499"}, "\x1f"),
		strings.Join([]string{"90", "coverage", "12000000", "2026-10-01T12:00:00Z", "1", "", "0"}, "\x1f"),
		strings.Join([]string{"80", "Weird Name", "2048", "2026-11-14T11:00:00Z", "0", "feat/tabs", "1"}, "\x1f"),
	}, "\n")
	arts, total := parseArtifacts(raw)
	if total != 4 || len(arts) != 4 {
		panic("parse artifacts")
	}
	caches := parseCaches(strings.Join([]string{
		strings.Join([]string{"7", "node-Linux-abc", "2000000000", "2026-11-01T00:00:00Z", "refs/heads/main", "2026-11-13T12:00:00Z"}, "\x1f"),
		strings.Join([]string{"8", "node-Linux-abc", "100000000", "2026-10-01T00:00:00Z", "refs/pull/12/merge", "2026-10-02T00:00:00Z"}, "\x1f"),
		strings.Join([]string{"9", "pip-Linux", "50000000", "2026-09-01T00:00:00Z", "refs/heads/main", "2026-09-02T00:00:00Z"}, "\x1f"),
	}, "\n"))
	p := Project{Owner: "acme", Name: "widgets", NWO: "acme/widgets", On: true}
	p.Items = append(arts, caches...)
	p.rebuildGroups()
	return p
}

func TestGroupsAndSelection(t *testing.T) {
	p := sampleProject()
	if p.ArtGroups[0] < 0 || p.Groups[p.ArtGroups[0]].Name != "dist" {
		t.Fatalf("largest artifact group = %s", p.Groups[p.ArtGroups[0]].Name)
	}
	if p.Groups[p.CacheGroups[0]].Name != "node-Linux-abc" {
		t.Fatalf("largest cache = %s", p.Groups[p.CacheGroups[0]].Name)
	}
	// Newest dist run is first inside the group.
	g := p.Groups[p.ArtGroups[0]]
	if p.Items[g.Start].ID != "101" || p.Items[g.Start+1].ID != "100" {
		t.Fatalf("order %s then %s", p.Items[g.Start].ID, p.Items[g.Start+1].ID)
	}
	p.toggleMatching(func(it Item) bool { return it.Group == p.ArtGroups[0] && it.matches("") })
	if !p.Items[g.Start].On || !p.Items[g.Start+1].On {
		t.Fatal("group toggle")
	}
	p.Items[g.Start+1].On = false
	live, on, _ := 0, 0, int64(0)
	live, on, _ = p.stateOf(func(it Item) bool { return it.Group == g.Start || it.Name == "dist" && it.Kind == "artifact" })
	if boxState(live, on) != "some" && on != 1 {
		t.Fatalf("partial live=%d on=%d", live, on)
	}
	n := p.selectExceptNewest(1, "")
	if n != 1 || p.itemOn("101") || !p.itemOn("100") || p.itemOn("90") {
		t.Fatalf("keep newest n=%d 101=%v 100=%v 90=%v", n, p.itemOn("101"), p.itemOn("100"), p.itemOn("90"))
	}
	if p.selectExpired("") != 1 || !p.itemOn("90") {
		t.Fatal("expired")
	}
	if p.Items[idIndex(p, "90")].Expired != true {
		t.Fatal("expired flag")
	}
	rows := p.rows("dist[1]", true, true)
	if len(rows) != 0 {
		t.Fatal("filter brackets should be literal")
	}
	rows = p.rows("release", true, true)
	foundCov := false
	for _, r := range rows {
		if r.Kind == rowGroup && p.Groups[r.Ref].Name == "coverage" {
			foundCov = true
		}
	}
	if foundCov {
		t.Fatal("branch filter showed coverage")
	}
}

func (p Project) itemOn(id string) bool {
	for _, it := range p.Items {
		if it.ID == id {
			return it.On
		}
	}
	return false
}

func idIndex(p Project, id string) int {
	for i, it := range p.Items {
		if it.ID == id {
			return i
		}
	}
	return -1
}

func TestDeleteOrder(t *testing.T) {
	p := sampleProject()
	m := newModel(options{wantArt: true, wantCache: true})
	m.projects = []Project{p}
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
	if stringsJoin(ids) != stringsJoin(want) {
		t.Fatalf("order %v", ids)
	}
}

func stringsJoin(s []string) string { return strings.Join(s, ",") }

func TestUISelectsProjectAndGroup(t *testing.T) {
	p := sampleProject()
	p.On = false
	other, _ := parseRepo("other/lib")
	m := newModel(options{wantArt: true, wantCache: true})
	m.ready = true
	m.projects = []Project{p, other}
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
	// Cursor starts on the first group. Space selects that whole group.
	next, _ = m.Update(teaKey(" "))
	m = next.(model)
	if !m.projects[0].itemOn("101") || !m.projects[0].itemOn("100") || m.projects[0].itemOn("90") {
		t.Fatal("space on the dist group should select both runs only")
	}
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
