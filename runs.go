package main

import (
	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
)

// runFilters are the filter flags of --delete-runs and --delete-run-logs.
var runFilters = []string{"older-than", "branch", "status"}

// runJobs are the two write flags of the workflow runs.
var runJobs = []job{
	{flag: "--delete-runs", verb: "delete", noun: "workflow runs", filters: runFilters, needsFilter: true,
		each: func(p project.Project, f project.Filter, dryRun bool) ([]string, int, error) {
			return deleteRuns(p, f, dryRun, "", gh.DeleteRun)
		}},
	{flag: "--delete-run-logs", verb: "delete", noun: "workflow run logs", filters: runFilters, needsFilter: true,
		each: func(p project.Project, f project.Filter, dryRun bool) ([]string, int, error) {
			return deleteRuns(p, f, dryRun, "logs", gh.DeleteRunLogs)
		}},
}

// deleteRuns reads the runs of one repository, selects the completed runs
// that pass the filter, and deletes each one, or its logs, with remove. It
// stops at the first error and gives the lines of the runs done before it.
func deleteRuns(p project.Project, f project.Filter, dryRun bool, what string, remove func(project.Project, string) error) ([]string, int, error) {
	runs, err := gh.ListRuns(p, f.Branch, f.Status)
	if err != nil {
		return nil, 0, err
	}
	var lines []string
	for _, r := range project.SelectRuns(runs, f, now()) {
		if !dryRun {
			if err := remove(p, r.ID); err != nil {
				return lines, len(lines), err
			}
		}
		lines = append(lines, project.RunLine(p.NWO, r, what, dryRun))
	}
	return lines, len(lines), nil
}
