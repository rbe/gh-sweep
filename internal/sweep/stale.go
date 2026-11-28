package sweep

import (
	"fmt"
	"time"

	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
)

var staleActions = []Action{
	{Flag: "--stale-list", Title: "List repositories by their last push", Verb: "print", Noun: "last push and disk usage",
		ReadOnly: true, Filters: []string{"older-than"}, TotalLine: project.StaleTotalLine, Account: staleEntries},
	{Flag: "--archive", Title: "Archive repositories", Verb: "archive", Noun: "repositories", Filters: []string{"older-than"},
		List: archiveEntries, Check: checkArchiveScope},
}

// staleEntries gives one line for each repository of the scope, oldest
// push first.
func staleEntries(s Scope, f project.Filter, now time.Time) ([]Entry, error) {
	states, err := repositoryStates(s)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for _, st := range project.SelectStale(states, f, now) {
		line := project.StaleLine(st, now)
		entries = append(entries, Entry{Line: func(bool) string { return line }, Total: project.Total{Count: 1, Bytes: st.Disk}})
	}
	return entries, nil
}

// repositoryStates reads the named repositories, or the list of --org,
// --user, or the logged-in user.
func repositoryStates(s Scope) ([]project.RepositoryState, error) {
	if len(s.Repos) == 0 {
		who := s.Org
		if who == "" {
			who = s.User
		}
		return gh.ListRepositoryStates(who)
	}
	repositories, err := parseNames(s.Repos)
	if err != nil {
		return nil, err
	}
	var states []project.RepositoryState
	for _, p := range repositories {
		st, err := gh.RepositoryStateOf(p)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p.NWO, err)
		}
		states = append(states, st)
	}
	return states, nil
}

// checkArchiveScope needs repository arguments. --mine, --org, and --user
// are an error, because an archive is for chosen repositories only.
func checkArchiveScope(s Scope, _ project.Filter) error {
	if len(s.Repos) == 0 || s.Mine || s.Org != "" || s.User != "" {
		return fmt.Errorf("--archive archives only the repositories you name. Name them, and do not pass --mine, --org, or --user")
	}
	return nil
}

// archiveEntries gives the entry of one repository. An archived
// repository has a line and no write. With --older-than, a repository
// pushed in the last N days is an error.
func archiveEntries(p project.Project, f project.Filter, now time.Time) ([]Entry, error) {
	st, err := gh.RepositoryStateOf(p)
	if err != nil {
		return nil, err
	}
	if !f.OlderThanCutoff(st.Pushed, now) {
		return nil, fmt.Errorf("pushed %s, less than %d days ago. Not archived", project.ArchiveDate(st), f.OlderThan)
	}
	if st.Archived {
		return []Entry{{Line: func(dry bool) string { return project.ArchiveLine(st, dry) }}}, nil
	}
	return []Entry{counted(func(dry bool) string { return project.ArchiveLine(st, dry) }, 0,
		func() error { return gh.Archive(p) })}, nil
}
