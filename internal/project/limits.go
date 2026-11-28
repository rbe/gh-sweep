package project

import (
	"time"

	"github.com/rbe/gh-sweep/internal/format"
)

// Large tells whether the item is an artifact of the large size of the
// preferences or more.
func (it Item) Large() bool {
	return it.Kind == "artifact" && it.Size >= Prefs().LargeArtifactBytes
}

// Old tells whether the item is an active artifact created more than the
// old days of the preferences before now.
func (it Item) Old(now time.Time) bool {
	if it.Kind != "artifact" || it.Expired || it.When.IsZero() {
		return false
	}
	return it.When.Before(now.Add(-time.Duration(Prefs().OldArtifactDays) * 24 * time.Hour))
}

// WithMinSize gives a copy of the inventories in which each repository keeps
// only the artifacts of minBytes or more. The caches stay. A minBytes of 0
// keeps every artifact.
func WithMinSize(inventories []Project, minBytes int64) []Project {
	if minBytes <= 0 {
		return inventories
	}
	out := make([]Project, 0, len(inventories))
	for _, p := range inventories {
		var kept []Item
		for _, it := range p.Items {
			if it.Kind != "artifact" || it.Size >= minBytes {
				kept = append(kept, it)
			}
		}
		p.Items = kept
		out = append(out, p)
	}
	return out
}

// now is the clock of the advice report. A test can replace format.Now.
func now() time.Time { return format.Now() }
