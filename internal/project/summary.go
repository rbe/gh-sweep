package project

import (
	"fmt"

	"github.com/rbe/gh-sweep/internal/format"
)

// SummaryLines is the text of the summary: one line for each repository
// that has an artifact or a cache, in the order given, then the total line.
// It reads the numbers of each inventory and changes nothing.
func SummaryLines(inventories []Project) []string {
	var lines []string
	var artifacts, caches int
	var bytes int64
	for _, p := range inventories {
		if p.ArtN == 0 && p.CacheN == 0 {
			continue
		}
		lines = append(lines, summaryLine(p.NWO, p.ArtN, p.CacheN, p.ArtBytes+p.CacheBytes))
		artifacts += p.ArtN
		caches += p.CacheN
		bytes += p.ArtBytes + p.CacheBytes
	}
	return append(lines, summaryLine("total", artifacts, caches, bytes))
}

func summaryLine(name string, artifacts, caches int, bytes int64) string {
	return fmt.Sprintf("%s: artifacts %d, caches %d, size %s", name, artifacts, caches, format.HumanSize(bytes))
}
