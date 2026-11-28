package project

import (
	"fmt"
	"sort"
	"time"

	"github.com/rbe/gh-sweep/internal/format"
)

// RepositoryState is what --stale-list and --archive read of a
// repository: the last push, the disk usage in bytes, and whether it is
// archived.
type RepositoryState struct {
	NWO      string
	Pushed   time.Time
	Disk     int64
	Archived bool
}

// SelectStale gives the repositories pushed more than OlderThan days ago,
// oldest push first. With no OlderThan it gives all of them.
func SelectStale(states []RepositoryState, f Filter, now time.Time) []RepositoryState {
	var out []RepositoryState
	for _, s := range states {
		if f.OlderThanCutoff(s.Pushed, now) {
			out = append(out, s)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Pushed.Before(out[j].Pushed) })
	return out
}

// StaleLine is the line of one repository after --stale-list.
func StaleLine(s RepositoryState, now time.Time) string {
	days := "-"
	if !s.Pushed.IsZero() {
		days = fmt.Sprint(int(now.Sub(s.Pushed).Hours() / 24))
	}
	return fmt.Sprintf("%s: pushed %s (%s days ago), disk %s, archived %s",
		s.NWO, format.Date(s.Pushed), days, format.HumanSize(s.Disk), yesNo(s.Archived))
}

// StaleTotalLine is the last line of --stale-list.
func StaleTotalLine(t Total, _ bool) string {
	return fmt.Sprintf("total: repositories %d, disk %s", t.Count, format.HumanSize(t.Bytes))
}

// ArchiveLine is the line of one repository after --archive.
func ArchiveLine(s RepositoryState, dryRun bool) string {
	action := ActionText("archive", dryRun)
	if s.Archived {
		action = "already archived"
	}
	return fmt.Sprintf("%s: pushed %s, disk %s, %s", s.NWO, format.Date(s.Pushed), format.HumanSize(s.Disk), action)
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// ArchiveDate is the date of the last push of the repository.
func ArchiveDate(s RepositoryState) string {
	return format.Date(s.Pushed)
}
