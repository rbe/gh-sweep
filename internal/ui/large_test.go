// id: ADR-0018

package ui

import (
	"strings"
	"testing"

	"github.com/rbe/gh-sweep/internal/project"
)

func TestItemRowMarksALargeArtifactAndNotASmallOne(t *testing.T) {
	m := newModel(Options{WantArt: true, WantCache: true})
	m.ready = true
	m.projects = []project.Project{sampleProject()}
	m.rebuildProj()
	m.width, m.height = 140, 40
	m.enterItems(0)
	next, _ := m.Update(teaKey("e"))
	view := stripANSI(next.(model).View())
	for _, line := range strings.Split(view, "\n") {
		large := strings.Contains(line, "· large")
		switch {
		case strings.Contains(line, "#101") && !large:
			t.Fatalf("the 900 MB artifact has no mark: %s", line)
		case strings.Contains(line, "#80") && large:
			t.Fatalf("the 2 KB artifact has a mark: %s", line)
		}
	}
	if !strings.Contains(view, "#101") || !strings.Contains(view, "#80") {
		t.Fatalf("view\n%s", view)
	}
}

func TestStyleMetaKeepsEveryPartOfTheRightSide(t *testing.T) {
	for _, right := range []string{"main · 2d · 900 MB · #101 · large · expires 2027-01-01", "main · 1h · 2 KB · #80", "#90 · expired"} {
		if got := stripANSI(styleMeta(right, false)); got != right {
			t.Fatalf("got %q, want %q", got, right)
		}
	}
}
