// id: ADR-0005

package project

import (
	"strings"
	"testing"
)

func TestUsageLinesListTheLargestRepositoryFirstWithItsShare(t *testing.T) {
	small := Project{NWO: "acme/small", Items: []Item{
		{Kind: "artifact", ID: "1", Name: "dist", Size: 1 << 20},
	}}
	small.RebuildGroups()
	large := Project{NWO: "acme/large", Items: []Item{
		{Kind: "artifact", ID: "2", Name: "dist", Size: 1 << 20, Expired: true},
		{Kind: "cache", ID: "3", Name: "node", Size: 2 << 20},
	}}
	large.RebuildGroups()
	empty := Project{NWO: "acme/empty"}
	empty.RebuildGroups()

	got := UsageLines([]Project{small, empty, large})
	want := []string{
		"acme/large: size 3.0 MB (3145728 bytes), artifacts 1.0 MB, expired artifacts 1.0 MB, caches 2.0 MB, share 75.0%",
		"acme/small: size 1.0 MB (1048576 bytes), artifacts 1.0 MB, expired artifacts 0 B, caches 0 B, share 25.0%",
		"total: size 4.0 MB (4194304 bytes), artifacts 2.0 MB, expired artifacts 1.0 MB, caches 2.0 MB",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("lines\n%s", strings.Join(got, "\n"))
	}
}

func TestUsageLinesPrintOnlyTheTotalWhenNoRepositoryUsesStorage(t *testing.T) {
	want := "total: size 0 B (0 bytes), artifacts 0 B, expired artifacts 0 B, caches 0 B"
	if got := UsageLines(nil); len(got) != 1 || got[0] != want {
		t.Fatalf("lines %q", got)
	}
}

func TestUsageLinesGiveAZeroShareWhenAllEntriesHaveNoBytes(t *testing.T) {
	zero := Project{NWO: "acme/zero", Items: []Item{{Kind: "artifact", ID: "1", Name: "dist"}}}
	zero.RebuildGroups()
	got := UsageLines([]Project{zero})
	if len(got) != 2 || !strings.HasSuffix(got[0], "share 0.0%") {
		t.Fatalf("lines %q", got)
	}
}
