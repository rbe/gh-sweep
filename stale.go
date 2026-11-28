package main

import (
	"fmt"

	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
	"github.com/rbe/gh-sweep/internal/ui"
)

// staleJobs are --stale-list and --archive.
var staleJobs = []job{
	{flag: "--stale-list", verb: "print", noun: "last push and disk usage", readOnly: true,
		filters: []string{"older-than"}, total: project.StaleTotalLine, account: staleList},
	{flag: "--archive", verb: "archive", noun: "repositories", filters: []string{"older-than"},
		each: archiveRepository, target: checkArchiveTarget},
}

// staleList prints the repositories of the target, oldest push first.
func staleList(o ui.Options, f project.Filter, _ bool) ([]string, project.Total, error) {
	states, err := repositoryStates(o)
	if err != nil {
		return nil, project.Total{}, err
	}
	var lines []string
	var t project.Total
	for _, s := range project.SelectStale(states, f, now()) {
		lines = append(lines, project.StaleLine(s, now()))
		t = t.Add(project.Total{Count: 1, Bytes: s.Disk})
	}
	return lines, t, nil
}

// repositoryStates reads the named repositories, or the list of --org,
// --user, or the logged-in user.
func repositoryStates(o ui.Options) ([]project.RepositoryState, error) {
	if len(o.Repos) == 0 {
		who := o.Org
		if who == "" {
			who = o.User
		}
		return gh.ListRepositoryStates(who)
	}
	repositories, err := parseRepositories(o.Repos)
	if err != nil {
		return nil, err
	}
	var states []project.RepositoryState
	for _, p := range repositories {
		s, err := gh.RepositoryStateOf(p)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p.NWO, err)
		}
		states = append(states, s)
	}
	return states, nil
}

// checkArchiveTarget needs repository arguments. --mine, --org, and --user
// are an error, because an archive is for chosen repositories only.
func checkArchiveTarget(o ui.Options, _ project.Filter) error {
	if len(o.Repos) == 0 || o.Mine || o.Org != "" || o.User != "" {
		return fmt.Errorf("--archive archives only the repositories you name. Name them, and do not pass --mine, --org, or --user")
	}
	return nil
}

// archiveRepository archives one repository. A repository that is already
// archived stays as it is. With --older-than, a repository pushed in the
// last N days is an error, so that a recent repository is not archived by
// a wrong name.
func archiveRepository(p project.Project, f project.Filter, dryRun bool) ([]string, project.Total, error) {
	s, err := gh.RepositoryStateOf(p)
	if err != nil {
		return nil, project.Total{}, err
	}
	if !f.OlderThanCutoff(s.Pushed, now()) {
		return nil, project.Total{}, fmt.Errorf("pushed %s, less than %d days ago. Not archived", project.ArchiveDate(s), f.OlderThan)
	}
	if s.Archived {
		return []string{project.ArchiveLine(s, dryRun)}, project.Total{}, nil
	}
	if !dryRun {
		if err := gh.Archive(p); err != nil {
			return nil, project.Total{}, err
		}
	}
	return []string{project.ArchiveLine(s, dryRun)}, project.Total{Count: 1}, nil
}
