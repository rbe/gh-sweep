package project

import (
	"fmt"
	"strings"
	"time"

	"github.com/rbe/gh-sweep/internal/format"
)

// SelectCaches gives the caches that --delete-caches takes: the key starts
// with the key prefix, the ref is the branch, and the last use is older
// than the age. An empty filter field selects every cache.
func SelectCaches(items []Item, f Filter, now time.Time) []Item {
	var out []Item
	for _, it := range items {
		if it.Kind != "cache" || !strings.HasPrefix(it.Name, f.KeyPrefix) {
			continue
		}
		if f.Branch != "" && it.Run != "refs/heads/"+f.Branch {
			continue
		}
		if !f.OlderThanCutoff(it.When, now) {
			continue
		}
		out = append(out, it)
	}
	return out
}

// CacheLine is the line of one cache after --delete-caches.
func CacheLine(nwo string, it Item, dryRun bool) string {
	return fmt.Sprintf("%s: cache %s %q, ref %s, size %s, last used %s, %s",
		nwo, it.ID, it.Name, dash(it.Meta), format.HumanSize(it.Size), format.Date(it.When), ActionText("delete", dryRun))
}

// CacheTotalLine is the last line of --delete-caches.
func CacheTotalLine(t Total, dryRun bool) string {
	return fmt.Sprintf("total: %s %d caches, size %s", ActionText("delete", dryRun), t.Count, format.HumanSize(t.Bytes))
}

// CacheUsage is the cache usage of a repository and its two cache limits.
// A limit of 0 is a limit that GitHub did not give.
type CacheUsage struct {
	Count         int
	Bytes         int64
	StorageGB     int
	RetentionDays int
}

// CacheUsageLine is the line of one repository after --cache-usage.
func CacheUsageLine(nwo string, u CacheUsage) string {
	storage, retention := "unknown", "unknown"
	if u.StorageGB > 0 {
		storage = fmt.Sprintf("%d GB", u.StorageGB)
	}
	if u.RetentionDays > 0 {
		retention = fmt.Sprintf("%d days", u.RetentionDays)
	}
	return fmt.Sprintf("%s: caches %d, size %s (%d bytes), storage limit %s, retention limit %s",
		nwo, u.Count, format.HumanSize(u.Bytes), u.Bytes, storage, retention)
}

// CacheUsageTotalLine is the last line of --cache-usage.
func CacheUsageTotalLine(t Total) string {
	return fmt.Sprintf("total: caches %d, size %s (%d bytes)", t.Count, format.HumanSize(t.Bytes), t.Bytes)
}
