package sweep

import (
	"fmt"
	"time"

	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
)

var cacheActions = []Action{
	{Flag: "--delete-caches", Title: "Delete caches", Verb: "delete", Noun: "caches", NeedsFilter: true,
		Filters: []string{"key-prefix", "branch", "older-than"}, TotalLine: project.CacheTotalLine, List: cacheEntries},
	{Flag: "--cache-usage", Title: "Show the cache usage and the cache limits", Verb: "print", Noun: "cache usage and cache limits",
		ReadOnly:  true,
		TotalLine: func(t project.Total, _ bool) string { return project.CacheUsageTotalLine(t) },
		List:      cacheUsageEntries},
}

// cacheEntries reads the caches of one repository and gives one entry for
// each cache that the filter selects. The write deletes the cache by its id.
func cacheEntries(p project.Project, f project.Filter, now time.Time) ([]Entry, error) {
	caches, err := gh.ListCaches(p)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for _, c := range project.SelectCaches(caches, f, now) {
		c := c
		entries = append(entries, counted(
			func(dry bool) string { return project.CacheLine(p.NWO, c, dry) }, c.Size,
			func() error { return gh.DeleteCache(p, c.ID) }))
	}
	return entries, nil
}

// cacheUsageEntries reads the cache usage and the two cache limits of one
// repository. A limit that GitHub refuses is unknown, the line still
// prints, and the refusal is the error of the repository.
func cacheUsageEntries(p project.Project, _ project.Filter, _ time.Time) ([]Entry, error) {
	u, err := gh.GetCacheUsage(p)
	if err != nil {
		return nil, err
	}
	var limitErr error
	if u.StorageGB, err = gh.GetCacheStorageLimit(p); err != nil {
		limitErr = fmt.Errorf("cache storage limit: %w", err)
	}
	if u.RetentionDays, err = gh.GetCacheRetentionLimit(p); err != nil && limitErr == nil {
		limitErr = fmt.Errorf("cache retention limit: %w", err)
	}
	line := project.CacheUsageLine(p.NWO, u)
	return []Entry{{Line: func(bool) string { return line }, Total: project.Total{Count: u.Count, Bytes: u.Bytes}}}, limitErr
}
