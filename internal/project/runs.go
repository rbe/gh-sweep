package project

import (
	"fmt"
	"time"

	"github.com/rbe/gh-sweep/internal/format"
)

// Run is one workflow run of a repository.
type Run struct {
	ID         string
	Name       string
	Branch     string
	Status     string
	Conclusion string
	Created    time.Time
}

// RunStatuses are the values that GitHub accepts for the status filter of
// the workflow runs: a status or a conclusion.
var RunStatuses = []string{
	"completed", "action_required", "cancelled", "failure", "neutral", "skipped", "stale",
	"success", "timed_out", "in_progress", "queued", "requested", "waiting", "pending",
}

// CheckRunStatus refuses a status that GitHub does not accept.
func CheckRunStatus(s string) error {
	if s == "" {
		return nil
	}
	for _, v := range RunStatuses {
		if v == s {
			return nil
		}
	}
	return fmt.Errorf("--status %s is not a status of a workflow run", s)
}

// SelectRuns gives the runs that a delete takes: only completed runs, and
// only runs older than the filter. GitHub already applied the branch and
// the status filter. A run that is not completed still runs or waits, and
// stays.
func SelectRuns(runs []Run, f Filter, now time.Time) []Run {
	var out []Run
	for _, r := range runs {
		if r.Status != "completed" || !f.OlderThanCutoff(r.Created, now) {
			continue
		}
		out = append(out, r)
	}
	return out
}

// RunLine is the line of one run after --delete-runs or --delete-run-logs.
// what is "" for the run, or "logs" for its logs only.
func RunLine(nwo string, r Run, what string, dryRun bool) string {
	action := ActionText("delete", dryRun)
	if what != "" {
		action += " " + what
	}
	return fmt.Sprintf("%s: run %s %q, branch %s, conclusion %s, created %s, %s",
		nwo, r.ID, r.Name, dash(r.Branch), dash(r.Conclusion), format.Date(r.Created), action)
}

// dash gives "-" for an empty value.
func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
