package project

import (
	"fmt"
	"strings"
)

// Branch is one branch of a repository: its name, the commit at its head,
// and whether a branch protection rule covers it.
type Branch struct {
	Name      string
	SHA       string
	Protected bool
}

// PullRequest is one pull request: its number, its state, whether it was
// merged, its head branch and commit, the repository of the head, and its
// base branch.
type PullRequest struct {
	Number   string
	State    string
	Merged   bool
	HeadRef  string
	HeadSHA  string
	HeadRepo string
	BaseRef  string
}

// StaleBranch is a branch that --delete-merged-branches takes, and the
// pull request that ended it.
type StaleBranch struct {
	Branch Branch
	PR     PullRequest
}

// SelectMergedBranches gives the branches of the repository nwo whose pull
// request is merged or closed. A branch stays when it is the default
// branch, when it is protected, when an open pull request has it as head or
// base, or when its head commit is not the head commit of the closed pull
// request, because it then has commits that no pull request holds.
func SelectMergedBranches(nwo, defaultBranch string, branches []Branch, pulls []PullRequest) []StaleBranch {
	open := map[string]bool{}
	ended := map[string]PullRequest{}
	for _, pr := range pulls {
		own := strings.EqualFold(pr.HeadRepo, nwo)
		if pr.State == "open" {
			open[pr.BaseRef] = true
			if own {
				open[pr.HeadRef] = true
			}
			continue
		}
		if own {
			if _, seen := ended[pr.HeadRef+"\x00"+pr.HeadSHA]; !seen {
				ended[pr.HeadRef+"\x00"+pr.HeadSHA] = pr
			}
		}
	}
	var out []StaleBranch
	for _, b := range branches {
		if b.Name == defaultBranch || b.Protected || open[b.Name] {
			continue
		}
		if pr, ok := ended[b.Name+"\x00"+b.SHA]; ok {
			out = append(out, StaleBranch{Branch: b, PR: pr})
		}
	}
	return out
}

// MergedBranchLine is the line of one branch after --delete-merged-branches.
func MergedBranchLine(nwo string, s StaleBranch, dryRun bool) string {
	how := "closed"
	if s.PR.Merged {
		how = "merged"
	}
	return fmt.Sprintf("%s: branch %s, pull request #%s %s, %s", nwo, s.Branch.Name, s.PR.Number, how, ActionText("delete", dryRun))
}
