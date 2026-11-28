package main

import (
	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
)

// branchJob is --delete-merged-branches.
var branchJob = job{
	flag: "--delete-merged-branches", verb: "delete", noun: "branches", each: deleteMergedBranches,
}

// deleteMergedBranches deletes the branches of one repository whose pull
// request is merged or closed. It stops at the first error.
func deleteMergedBranches(p project.Project, _ project.Filter, dryRun bool) ([]string, project.Total, error) {
	defaultBranch, err := gh.DefaultBranch(p)
	if err != nil {
		return nil, project.Total{}, err
	}
	branches, err := gh.ListBranches(p)
	if err != nil {
		return nil, project.Total{}, err
	}
	pulls, err := gh.ListPullRequests(p)
	if err != nil {
		return nil, project.Total{}, err
	}
	var lines []string
	for _, s := range project.SelectMergedBranches(p.Owner+"/"+p.Name, defaultBranch, branches, pulls) {
		if !dryRun {
			if err := gh.DeleteBranch(p, s.Branch.Name); err != nil {
				return lines, project.Total{Count: len(lines)}, err
			}
		}
		lines = append(lines, project.MergedBranchLine(p.NWO, s, dryRun))
	}
	return lines, project.Total{Count: len(lines)}, nil
}
