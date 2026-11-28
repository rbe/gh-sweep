package sweep

import (
	"time"

	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
)

var branchAction = Action{
	Flag: "--delete-merged-branches", Title: "Delete branches of merged or closed pull requests", Verb: "delete", Noun: "branches",
	List: branchEntries,
}

// branchEntries gives one entry for each branch whose pull request is
// merged or closed.
func branchEntries(p project.Project, _ project.Filter, _ time.Time) ([]Entry, error) {
	defaultBranch, err := gh.DefaultBranch(p)
	if err != nil {
		return nil, err
	}
	branches, err := gh.ListBranches(p)
	if err != nil {
		return nil, err
	}
	pulls, err := gh.ListPullRequests(p)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for _, s := range project.SelectMergedBranches(p.Owner+"/"+p.Name, defaultBranch, branches, pulls) {
		s := s
		entries = append(entries, counted(
			func(dry bool) string { return project.MergedBranchLine(p.NWO, s, dry) }, 0,
			func() error { return gh.DeleteBranch(p, s.Branch.Name) }))
	}
	return entries, nil
}
