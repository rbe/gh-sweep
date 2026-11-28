package project

import (
	"fmt"
	"sort"

	"github.com/rbe/gh-sweep/internal/format"
)

// UsageLines is the text of the storage usage breakdown: one line for each
// repository that uses storage, largest first, then the total line. A line
// gives the size in bytes, the artifacts, the expired artifacts, the caches,
// and the share of the total. It reads the numbers and changes nothing.
func UsageLines(inventories []Project) []string {
	used := usedRepositories(inventories)
	var all usage
	for _, u := range used {
		all.add(u)
	}
	lines := make([]string, 0, len(used)+1)
	for _, u := range used {
		lines = append(lines, u.line(all.total()))
	}
	return append(lines, all.totalLine())
}

// usage is the storage of one repository, or of all repositories.
type usage struct {
	name      string
	artifacts int64
	expired   int64
	caches    int64
}

func (u usage) total() int64 { return u.artifacts + u.caches }

func (u *usage) add(other usage) {
	u.artifacts += other.artifacts
	u.expired += other.expired
	u.caches += other.caches
}

// usedRepositories keeps each repository with a byte of storage, and sorts
// them by size, largest first. Two of the same size keep the order of names.
func usedRepositories(inventories []Project) []usage {
	var used []usage
	for _, p := range inventories {
		if p.ArtN == 0 && p.CacheN == 0 {
			continue
		}
		used = append(used, usage{name: p.NWO, artifacts: p.ArtBytes, expired: p.ExpBytes, caches: p.CacheBytes})
	}
	sort.SliceStable(used, func(a, b int) bool {
		if used[a].total() != used[b].total() {
			return used[a].total() > used[b].total()
		}
		return used[a].name < used[b].name
	})
	return used
}

func (u usage) line(all int64) string {
	return fmt.Sprintf("%s: %s, share %s", u.name, u.parts(), share(u.total(), all))
}

func (u usage) totalLine() string {
	return "total: " + u.parts()
}

func (u usage) parts() string {
	return fmt.Sprintf("size %s (%d bytes), artifacts %s, expired artifacts %s, caches %s",
		format.HumanSize(u.total()), u.total(),
		format.HumanSize(u.artifacts), format.HumanSize(u.expired), format.HumanSize(u.caches))
}

// share is part as a percentage of all, with one decimal.
func share(part, all int64) string {
	if all <= 0 {
		return "0.0%"
	}
	return fmt.Sprintf("%.1f%%", float64(part)*100/float64(all))
}
