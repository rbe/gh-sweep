package project

import (
	"fmt"
	"sort"

	"github.com/rbe/gh-sweep/internal/format"
)

// ArtifactLines is the text of the artifact list: one line for each
// artifact, then the total line. The repositories keep the order given, and
// the artifacts of one repository stand largest first. A line gives the
// size, the creation date, the expiration date, and the status. Dates are
// UTC. It reads the items and changes nothing.
func ArtifactLines(inventories []Project) []string {
	var lines []string
	var count, expired int
	var bytes, expiredBytes int64
	for _, p := range inventories {
		for _, it := range artifactsBySize(p.Items) {
			lines = append(lines, artifactLine(p.NWO, it))
			count++
			bytes += it.Size
			if it.Expired {
				expired++
				expiredBytes += it.Size
			}
		}
	}
	total := fmt.Sprintf("total: artifacts %d, size %s, expired %d, expired size %s",
		count, format.HumanSize(bytes), expired, format.HumanSize(expiredBytes))
	return append(lines, total)
}

// artifactsBySize keeps the live artifacts and sorts them largest first.
// Two of the same size stand by name, then by id.
func artifactsBySize(items []Item) []Item {
	var arts []Item
	for _, it := range items {
		if it.Kind == "artifact" && !it.Dead {
			arts = append(arts, it)
		}
	}
	sort.SliceStable(arts, func(a, b int) bool {
		if arts[a].Size != arts[b].Size {
			return arts[a].Size > arts[b].Size
		}
		if arts[a].Name != arts[b].Name {
			return arts[a].Name < arts[b].Name
		}
		return arts[a].ID < arts[b].ID
	})
	return arts
}

func artifactLine(nwo string, it Item) string {
	return fmt.Sprintf("%s: %s #%s, size %s, created %s, expires %s, status %s",
		nwo, it.Name, it.ID, format.HumanSize(it.Size),
		format.Date(it.When), format.Date(it.Expires), artifactStatus(it))
}

func artifactStatus(it Item) string {
	if it.Expired {
		return "expired"
	}
	return "active"
}
