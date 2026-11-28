// id: ADR-0005

package project

import (
	"strings"
	"testing"
	"time"
)

func TestArtifactLinesGiveSizeDatesAndStatusLargestFirst(t *testing.T) {
	created := time.Date(2026, 11, 10, 23, 30, 0, 0, time.FixedZone("CET", 3600))
	expires := time.Date(2027, 2, 8, 12, 0, 0, 0, time.UTC)
	widgets := Project{NWO: "acme/widgets", Items: []Item{
		{Kind: "artifact", ID: "1", Name: "logs", Size: 512, When: created, Expires: expires, Expired: true},
		{Kind: "artifact", ID: "2", Name: "dist", Size: 1 << 20, When: created, Expires: expires},
		{Kind: "artifact", ID: "3", Name: "gone", Size: 1 << 30, Dead: true},
		{Kind: "cache", ID: "4", Name: "node", Size: 2 << 20},
	}}
	widgets.RebuildGroups()
	lib := Project{NWO: "other/lib", Items: []Item{
		{Kind: "artifact", ID: "9", Name: "wheel", Size: 2048},
	}}
	lib.RebuildGroups()

	got := ArtifactLines([]Project{widgets, lib})
	want := []string{
		"acme/widgets: dist #2, size 1.0 MB, created 2026-11-10, expires 2027-02-08, status active",
		"acme/widgets: logs #1, size 512 B, created 2026-11-10, expires 2027-02-08, status expired",
		"other/lib: wheel #9, size 2.0 KB, created -, expires -, status active",
		"total: artifacts 3, size 1.0 MB, expired 1, expired size 512 B",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("lines\n%s", strings.Join(got, "\n"))
	}
}

func TestArtifactLinesPrintOnlyTheTotalWhenNoRepositoryHasAnArtifact(t *testing.T) {
	want := "total: artifacts 0, size 0 B, expired 0, expired size 0 B"
	if got := ArtifactLines(nil); len(got) != 1 || got[0] != want {
		t.Fatalf("lines %q", got)
	}
}
