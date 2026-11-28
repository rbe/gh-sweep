package project

import (
	"time"

	"github.com/rbe/gh-sweep/internal/format"
)

// LargeArtifactBytes is the size from which an artifact is large: 50 MB.
const LargeArtifactBytes int64 = 50 << 20

// OldArtifactDays is the age in days after which an active artifact is old.
const OldArtifactDays = 5

// Large tells whether the item is an artifact of LargeArtifactBytes or more.
func (it Item) Large() bool {
	return it.Kind == "artifact" && it.Size >= LargeArtifactBytes
}

// Old tells whether the item is an active artifact created more than
// OldArtifactDays days before now.
func (it Item) Old(now time.Time) bool {
	if it.Kind != "artifact" || it.Expired || it.When.IsZero() {
		return false
	}
	return it.When.Before(now.Add(-OldArtifactDays * 24 * time.Hour))
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
