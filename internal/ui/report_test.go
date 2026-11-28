// id: ADR-0005

package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/rbe/gh-sweep/internal/project"
)

func TestUOnTheProjectListShowsTheUsageOfTheScannedProjects(t *testing.T) {
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
	m.width, m.height = 200, 36

	next, _ := m.Update(teaKey("u"))
	m = next.(model)
	view := stripANSI(m.View())
	for _, want := range []string{
		"storage usage",
		"1 projects are not scanned",
		"acme/widgets: size 3.4 GB (3662002048 bytes), artifacts 1.4 GB, expired artifacts 11.4 MB, caches 2.0 GB, share 100.0%",
		"total: size 3.4 GB",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("no %q in\n%s", want, view)
		}
	}
	next, _ = m.Update(teaKey("b"))
	if next.(model).screen != scrProjects {
		t.Fatalf("b goes to screen %d", next.(model).screen)
	}
}

func TestIInsideAProjectShowsEachArtifactWithItsDatesAndStatus(t *testing.T) {
	m := newModel(Options{WantArt: true, WantCache: true})
	m.ready = true
	m.projects = []project.Project{sampleProject()}
	m.projects[0].Items[idIndex(m.projects[0], "101")].Expires = time.Date(2027, 2, 10, 12, 0, 0, 0, time.UTC)
	m.width, m.height = 200, 36
	m.enterItems(0)
	if view := stripANSI(m.View()); !strings.Contains(view, "expires 2027-02-10") {
		t.Fatalf("the item row has no expiration date\n%s", view)
	}

	next, _ := m.Update(teaKey("i"))
	m = next.(model)
	view := stripANSI(m.View())
	for _, want := range []string{
		"artifacts of acme/widgets",
		"acme/widgets: dist #101, size 858 MB, created 2026-11-12, expires 2027-02-10, status active",
		"acme/widgets: coverage #90, size 11.4 MB, created 2026-10-01, expires -, status expired",
		"total: artifacts 4, size 1.4 GB, expired 1, expired size 11.4 MB",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("no %q in\n%s", want, view)
		}
	}
	next, _ = m.Update(teaKey("esc"))
	if next.(model).screen != scrItems {
		t.Fatalf("esc goes to screen %d", next.(model).screen)
	}
}

func TestTheReportScreenScrollsWithinItsLines(t *testing.T) {
	m := newModel(Options{WantArt: true})
	m.height = 8
	m.showReport("t", "", []string{"1", "2", "3", "4", "5"})
	for i := 0; i < 9; i++ {
		next, _ := m.Update(teaKey("j"))
		m = next.(model)
	}
	if m.report.scroll != 3 {
		t.Fatalf("scroll %d after j", m.report.scroll)
	}
	for i := 0; i < 9; i++ {
		next, _ := m.Update(teaKey("k"))
		m = next.(model)
	}
	if m.report.scroll != 0 {
		t.Fatalf("scroll %d after k", m.report.scroll)
	}
}

func idIndex(p project.Project, id string) int {
	for i, it := range p.Items {
		if it.ID == id {
			return i
		}
	}
	return -1
}
