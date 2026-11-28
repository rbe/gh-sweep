package gh

import (
	"fmt"
	"strings"

	"github.com/rbe/gh-sweep/internal/project"
)

// ListCaches reads the caches of the repository, largest first.
func ListCaches(p project.Project) ([]project.Item, error) {
	raw, err := fetchCaches(p)
	if err != nil {
		return nil, err
	}
	return ParseCaches(raw), nil
}

// GetCacheUsage reads the number and the size of the active caches.
func GetCacheUsage(p project.Project) (project.CacheUsage, error) {
	args := append(apiArgs(p), fmt.Sprintf("repos/%s/%s/actions/cache/usage", p.Owner, p.Name),
		"--jq", `"\(.active_caches_count // 0) \(.active_caches_size_in_bytes // 0)"`)
	out, err := runGHFunc(args...)
	if err != nil {
		return project.CacheUsage{}, err
	}
	f := strings.Fields(out)
	if len(f) != 2 {
		return project.CacheUsage{}, fmt.Errorf("gh gave no cache usage: %q", strings.TrimSpace(out))
	}
	return project.CacheUsage{Count: int(atoi64(f[0])), Bytes: atoi64(f[1])}, nil
}

// GetCacheStorageLimit reads the largest total cache size in gigabytes.
func GetCacheStorageLimit(p project.Project) (int, error) {
	return getNumber(p, "repos/%s/%s/actions/cache/storage-limit", ".max_cache_size_gb // 0")
}

// GetCacheRetentionLimit reads the number of days that GitHub keeps a cache.
func GetCacheRetentionLimit(p project.Project) (int, error) {
	return getNumber(p, "repos/%s/%s/actions/cache/retention-limit", ".max_cache_retention_days // 0")
}

// getNumber reads one number of a repository endpoint.
func getNumber(p project.Project, path, jq string) (int, error) {
	out, err := runGHFunc(append(apiArgs(p), fmt.Sprintf(path, p.Owner, p.Name), "--jq", jq)...)
	if err != nil {
		return 0, err
	}
	return int(atoi64(out)), nil
}
