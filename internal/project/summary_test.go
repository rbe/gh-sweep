// id: ADR-0003

package project

import (
	"strings"
	"testing"
)

func TestSummaryLinesHasOneLineForEachRepositoryThenTheTotal(t *testing.T) {
	widgets := Project{NWO: "acme/widgets", Items: []Item{
		{Kind: "artifact", ID: "1", Name: "dist", Size: 1 << 20},
		{Kind: "artifact", ID: "2", Name: "dist", Size: 1 << 19},
		{Kind: "cache", ID: "7", Name: "node", Size: 2 << 20},
	}}
	widgets.RebuildGroups()
	lib := Project{NWO: "other/lib", Items: []Item{
		{Kind: "cache", ID: "8", Name: "pip", Size: 512},
	}}
	lib.RebuildGroups()

	got := SummaryLines([]Project{widgets, lib})
	want := []string{
		"acme/widgets: artifacts 2, caches 1, size 3.5 MB",
		"other/lib: artifacts 0, caches 1, size 512 B",
		"total: artifacts 2, caches 2, size 3.5 MB",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("lines\n%s", strings.Join(got, "\n"))
	}
}

func TestSummaryLinesPutsTheTotalLast(t *testing.T) {
	empty := Project{NWO: "acme/empty"}
	empty.RebuildGroups()
	got := SummaryLines([]Project{empty})
	if len(got) != 1 || got[0] != "total: artifacts 0, caches 0, size 0 B" {
		t.Fatalf("lines %q", got)
	}
	if got := SummaryLines(nil); len(got) != 1 || got[0] != "total: artifacts 0, caches 0, size 0 B" {
		t.Fatalf("no inventory %q", got)
	}
}
