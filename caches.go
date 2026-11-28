package main

import (
	"fmt"

	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
)

// cacheJobs are the flags of the caches.
var cacheJobs = []job{
	{flag: "--delete-caches", verb: "delete", noun: "caches", needsFilter: true,
		filters: []string{"key-prefix", "branch", "older-than"},
		total:   project.CacheTotalLine,
		each:    deleteCaches},
	{flag: "--cache-usage", verb: "print", noun: "cache usage and cache limits", readOnly: true,
		total: func(t project.Total, _ bool) string { return project.CacheUsageTotalLine(t) },
		each:  cacheUsage},
}

// deleteCaches reads the caches of one repository, selects them with the
// filter, and deletes each one by its id. It stops at the first error and
// gives the lines of the caches done before it.
func deleteCaches(p project.Project, f project.Filter, dryRun bool) ([]string, project.Total, error) {
	caches, err := gh.ListCaches(p)
	if err != nil {
		return nil, project.Total{}, err
	}
	var lines []string
	var t project.Total
	for _, c := range project.SelectCaches(caches, f, now()) {
		if !dryRun {
			if err := gh.DeleteCache(p, c.ID); err != nil {
				return lines, t, err
			}
		}
		lines = append(lines, project.CacheLine(p.NWO, c, dryRun))
		t = t.Add(project.Total{Count: 1, Bytes: c.Size})
	}
	return lines, t, nil
}

// cacheUsage reads the cache usage and the two cache limits of one
// repository. A limit that GitHub refuses is unknown, the line still
// prints, and the refusal is the error of the repository.
func cacheUsage(p project.Project, _ project.Filter, _ bool) ([]string, project.Total, error) {
	u, err := gh.GetCacheUsage(p)
	if err != nil {
		return nil, project.Total{}, err
	}
	var limitErr error
	if u.StorageGB, err = gh.GetCacheStorageLimit(p); err != nil {
		limitErr = fmt.Errorf("cache storage limit: %w", err)
	}
	if u.RetentionDays, err = gh.GetCacheRetentionLimit(p); err != nil && limitErr == nil {
		limitErr = fmt.Errorf("cache retention limit: %w", err)
	}
	return []string{project.CacheUsageLine(p.NWO, u)}, project.Total{Count: u.Count, Bytes: u.Bytes}, limitErr
}
