package project

import (
	"fmt"
	"time"
)

// Filter is what a write flag selects in a repository. A zero field
// selects everything. OlderThan is a number of days, Keep is a number of
// newest entries that stay.
type Filter struct {
	OlderThan   int
	Branch      string
	Status      string
	KeyPrefix   string
	Keep        int
	Untagged    bool
	Package     string
	PackageType string
}

// Total is what a job counted in one repository or in all of them: the
// entries and their bytes.
type Total struct {
	Count int
	Bytes int64
	Gross float64
	Net   float64
}

// Add gives the sum of two totals.
func (t Total) Add(o Total) Total {
	return Total{Count: t.Count + o.Count, Bytes: t.Bytes + o.Bytes, Gross: t.Gross + o.Gross, Net: t.Net + o.Net}
}

// OlderThanCutoff is true when t is more than OlderThan days before now.
// With no OlderThan every time passes. An unknown time never passes when
// OlderThan is set.
func (f Filter) OlderThanCutoff(t, now time.Time) bool {
	if f.OlderThan == 0 {
		return true
	}
	return !t.IsZero() && t.Before(now.AddDate(0, 0, -f.OlderThan))
}

// pastTense is the word of a done write.
var pastTense = map[string]string{
	"delete":  "deleted",
	"archive": "archived",
	"change":  "changed",
}

// ActionText says that a write would happen with --dry-run, or that it
// happened: "would delete" or "deleted".
func ActionText(verb string, dryRun bool) string {
	if dryRun {
		return "would " + verb
	}
	return pastTense[verb]
}

// TotalLine is the last line of a write flag: "total: would delete 3 runs".
func TotalLine(verb, noun string, n int, dryRun bool) string {
	return fmt.Sprintf("total: %s %d %s", ActionText(verb, dryRun), n, noun)
}
