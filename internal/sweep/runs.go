package sweep

import (
	"time"

	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
)

// runFilters are the filters of the two run actions.
var runFilters = []string{"older-than", "branch", "status"}

var runActions = []Action{
	{Flag: "--delete-runs", Title: "Delete workflow runs and their logs", Verb: "delete", Noun: "workflow runs",
		Filters: runFilters, NeedsFilter: true,
		List: func(p project.Project, f project.Filter, now time.Time) ([]Entry, error) {
			return runEntries(p, f, now, "", gh.DeleteRun)
		}},
	{Flag: "--delete-run-logs", Title: "Delete the logs of workflow runs", Verb: "delete", Noun: "workflow run logs",
		Filters: runFilters, NeedsFilter: true,
		List: func(p project.Project, f project.Filter, now time.Time) ([]Entry, error) {
			return runEntries(p, f, now, "logs", gh.DeleteRunLogs)
		}},
}

// runEntries reads the runs of one repository and gives one entry for each
// completed run that passes the filter. The write deletes the run, or its
// logs, with remove.
func runEntries(p project.Project, f project.Filter, now time.Time, what string, remove func(project.Project, string) error) ([]Entry, error) {
	runs, err := gh.ListRuns(p, f.Branch, f.Status)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for _, r := range project.SelectRuns(runs, f, now) {
		r := r
		entries = append(entries, counted(
			func(dry bool) string { return project.RunLine(p.NWO, r, what, dry) }, 0,
			func() error { return remove(p, r.ID) }))
	}
	return entries, nil
}
