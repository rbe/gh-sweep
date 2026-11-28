package project

import (
	"fmt"
	"sort"
	"time"

	"github.com/rbe/gh-sweep/internal/format"
)

// Release is one release of a repository and its assets.
type Release struct {
	ID        string
	Name      string
	Tag       string
	Draft     bool
	Immutable bool
	Created   time.Time
	Assets    []Asset
}

// Asset is one file of a release.
type Asset struct {
	ID   string
	Name string
	Size int64
}

// Bytes is the size of all assets of the release.
func (r Release) Bytes() int64 {
	var n int64
	for _, a := range r.Assets {
		n += a.Size
	}
	return n
}

// SelectReleases gives the releases that a delete takes: newest first,
// without the newest Keep, and only releases created more than OlderThan
// days ago.
func SelectReleases(releases []Release, f Filter, now time.Time) []Release {
	sorted := append([]Release{}, releases...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Created.After(sorted[j].Created) })
	var out []Release
	for i, r := range sorted {
		if i < f.Keep || !f.OlderThanCutoff(r.Created, now) {
			continue
		}
		out = append(out, r)
	}
	return out
}

// ReleaseLine is the line of one release after --delete-releases.
func ReleaseLine(nwo string, r Release, dryRun bool) string {
	return fmt.Sprintf("%s: release %s %q, tag %s, created %s, assets %d, size %s, %s",
		nwo, r.ID, r.Name, dash(r.Tag), format.Date(r.Created), len(r.Assets), format.HumanSize(r.Bytes()), ActionText("delete", dryRun))
}

// AssetLine is the line of one asset after --delete-release-assets.
func AssetLine(nwo string, r Release, a Asset, dryRun bool) string {
	return fmt.Sprintf("%s: release %s asset %s %q, size %s, %s",
		nwo, dash(r.Tag), a.ID, a.Name, format.HumanSize(a.Size), ActionText("delete", dryRun))
}

// SizeTotalLine is a total line with a number and a size:
// "total: would delete 3 releases, size 1.0 MB".
func SizeTotalLine(noun string) func(Total, bool) string {
	return func(t Total, dryRun bool) string {
		return fmt.Sprintf("total: %s %d %s, size %s", ActionText("delete", dryRun), t.Count, noun, format.HumanSize(t.Bytes))
	}
}
