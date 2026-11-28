// id: ADR-0018

package project

import (
	"strings"
	"testing"
	"time"

	"github.com/rbe/gh-sweep/internal/format"
)

func adviceInventory(t *testing.T) []Project {
	t.Helper()
	format.Now = func() time.Time { return time.Date(2026, 12, 31, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { format.Now = time.Now })
	big := Project{NWO: "acme/big", Items: []Item{
		{Kind: "artifact", ID: "1", Name: "dist", Size: 60 << 20, When: time.Date(2026, 12, 20, 0, 0, 0, 0, time.UTC)},
		{Kind: "artifact", ID: "2", Name: "dist", Size: 2 << 20, When: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)},
		{Kind: "artifact", ID: "3", Name: "logs", Size: 1 << 20, When: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), Expired: true},
	}}
	big.Recompute()
	small := Project{NWO: "acme/small", Items: []Item{{Kind: "cache", ID: "7", Name: "node", Size: 1 << 10}}}
	small.Recompute()
	return []Project{small, big}
}

func TestAdviceNamesTheLargestRepositoriesAndTheExpiredOldAndLargeArtifacts(t *testing.T) {
	got := strings.Join(AdviceLines(adviceInventory(t)), "\n")
	want := strings.Join([]string{
		"1. Largest repositories: acme/big (63.0 MB), acme/small (1.0 KB).",
		"2. Expired artifacts: 1, size 1.0 MB. They still count against the quota. In a repository of the checklist, x selects them.",
		"3. Active artifacts older than 30 days: 1, size 2.0 MB. In a repository of the checklist, o selects them.",
		"4. Artifacts of 50.0 MB or more: 1, size 60.0 MB. Upload fewer files, or set a shorter retention with --retention-days.",
		"total: repositories 2, size 63.0 MB",
	}, "\n")
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

func TestAdviceTableHasOneRowForEachFindingAndTheTotal(t *testing.T) {
	table := AdviceTable(adviceInventory(t))
	if len(table.Rows) != 4 || table.Rows[1][0] != "expired" || table.Rows[1][2] != int64(1<<20) {
		t.Fatalf("rows %v", table.Rows)
	}
	if table.Total[0] != 2 || table.Total[1] != int64(63<<20+1<<10) {
		t.Fatalf("total %v", table.Total)
	}
}

func TestAdviceSaysNoAdviceWhenNoRepositoryUsesStorage(t *testing.T) {
	got := AdviceLines(nil)
	if len(got) != 2 || got[0] != "no advice: no expired, old, or large artifact, and no storage" || got[1] != "total: repositories 0, size 0 B" {
		t.Fatalf("got %q", got)
	}
	if rows := AdviceTable(nil).Rows; len(rows) != 1 || rows[0][0] != "none" {
		t.Fatalf("rows %v", rows)
	}
}

func TestWithMinSizeKeepsTheLargerArtifactsAndEveryCache(t *testing.T) {
	inv := adviceInventory(t)
	kept := WithMinSize(inv, 2<<20)
	if len(kept[0].Items) != 1 || len(kept[1].Items) != 2 || kept[1].Items[1].ID != "2" {
		t.Fatalf("kept %v", kept)
	}
	if len(inv[1].Items) != 3 {
		t.Fatal("WithMinSize changed its input")
	}
	if got := WithMinSize(inv, 0); len(got[1].Items) != 3 {
		t.Fatal("a minimum of 0 dropped an artifact")
	}
}

func TestTopUsageLinesShowsTheLargestRepositoriesAndCountsAllInTheTotal(t *testing.T) {
	got := TopUsageLines(adviceInventory(t), 1)
	if len(got) != 2 || !strings.HasPrefix(got[0], "acme/big: ") || !strings.HasSuffix(got[0], "share 100.0%") ||
		!strings.HasPrefix(got[1], "total: size 63.0 MB (66061312 bytes)") {
		t.Fatalf("got %q", got)
	}
	if rows := TopUsageTable(adviceInventory(t), 1).Rows; len(rows) != 1 {
		t.Fatalf("rows %v", rows)
	}
}

func TestParseSizeReadsBytesAndUnitsAndRefusesOtherText(t *testing.T) {
	for text, want := range map[string]int64{"0": 0, "1048576": 1 << 20, "50MB": 50 << 20, "2 kb": 2 << 10, "1GB": 1 << 30, "7B": 7} {
		if got, err := ParseSize(text); err != nil || got != want {
			t.Fatalf("%q: got %d, %v", text, got, err)
		}
	}
	for _, text := range []string{"", "-1", "1.5MB", "5XB", "MB"} {
		if _, err := ParseSize(text); err == nil {
			t.Fatalf("%q gave no error", text)
		}
	}
}
