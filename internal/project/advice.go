package project

import (
	"fmt"
	"strings"

	"github.com/rbe/gh-sweep/internal/format"
)

// advice is one finding of the advice report: its kind, the count and the
// bytes it is about, and its text.
type advice struct {
	kind  string
	count int
	bytes int64
	text  string
}

// adviceTopRepositories is the number of repositories the first advice names.
const adviceTopRepositories = 3

// AdviceLines is the text of the advice report: one numbered line for each
// finding, or one line when there is none, then the total line.
func AdviceLines(inventories []Project) []string {
	found := adviceOf(inventories)
	lines := make([]string, 0, len(found)+1)
	for i, a := range found {
		if a.kind == "none" {
			lines = append(lines, a.text)
			continue
		}
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, a.text))
	}
	repositories, size := adviceTotal(inventories)
	return append(lines, fmt.Sprintf("total: repositories %d, size %s", repositories, format.HumanSize(size)))
}

// AdviceTable is the advice report as data, with the rows of AdviceLines.
func AdviceTable(inventories []Project) Table {
	t := Table{Report: "advice", Columns: []string{"advice", "count", "bytes", "text"},
		TotalColumns: []string{"repositories", "bytes"}, Rows: [][]any{}}
	for _, a := range adviceOf(inventories) {
		t.Rows = append(t.Rows, []any{a.kind, a.count, a.bytes, a.text})
	}
	repositories, size := adviceTotal(inventories)
	t.Total = []any{repositories, size}
	return t
}

// adviceTotal counts the read repositories and the bytes they use.
func adviceTotal(inventories []Project) (int, int64) {
	var size int64
	for _, p := range inventories {
		size += p.ArtBytes + p.CacheBytes
	}
	return len(inventories), size
}

// adviceOf gives the findings in their fixed order, without a finding of
// count 0. With no finding it gives the one finding of the kind none.
func adviceOf(inventories []Project) []advice {
	var found []advice
	if a, ok := largestRepositories(inventories); ok {
		found = append(found, a)
	}
	expired, old, large := artifactFindings(inventories)
	for _, a := range []advice{expired, old, large} {
		if a.count > 0 {
			found = append(found, a)
		}
	}
	if len(found) == 0 {
		return []advice{{kind: "none", text: "no advice: no expired, old, or large artifact, and no storage"}}
	}
	return found
}

// largestRepositories names the largest repositories with their sizes.
func largestRepositories(inventories []Project) (advice, bool) {
	used, _ := usageOf(inventories, adviceTopRepositories)
	if len(used) == 0 {
		return advice{}, false
	}
	names := make([]string, 0, len(used))
	var size int64
	for _, u := range used {
		names = append(names, fmt.Sprintf("%s (%s)", u.name, format.HumanSize(u.total())))
		size += u.total()
	}
	text := "Largest repositories: " + strings.Join(names, ", ") + "."
	return advice{kind: "largest-repositories", count: len(used), bytes: size, text: text}, true
}

// artifactFindings counts the expired, the old, and the large artifacts.
func artifactFindings(inventories []Project) (expired, old, large advice) {
	at := now()
	for _, p := range inventories {
		for _, it := range p.Items {
			if it.Kind != "artifact" || it.Dead {
				continue
			}
			if it.Expired {
				expired.count++
				expired.bytes += it.Size
			}
			if it.Old(at) {
				old.count++
				old.bytes += it.Size
			}
			if it.Large() {
				large.count++
				large.bytes += it.Size
			}
		}
	}
	expired.kind, old.kind, large.kind = "expired", "old", "large"
	expired.text = fmt.Sprintf("Expired artifacts: %d, size %s. They still count against the quota. In a repository of the checklist, x selects them.",
		expired.count, format.HumanSize(expired.bytes))
	old.text = fmt.Sprintf("Active artifacts older than %d days: %d, size %s. In a repository of the checklist, o selects them.",
		OldArtifactDays, old.count, format.HumanSize(old.bytes))
	large.text = fmt.Sprintf("Artifacts of %s or more: %d, size %s. Upload fewer files, or set a shorter retention with --retention-days.",
		format.HumanSize(LargeArtifactBytes), large.count, format.HumanSize(large.bytes))
	return expired, old, large
}
