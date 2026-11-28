package project

import (
	"fmt"
	"time"

	"github.com/rbe/gh-sweep/internal/format"
)

// Codespace is one codespace of the logged-in user.
type Codespace struct {
	Name        string
	Repository  string
	State       string
	LastUsed    time.Time
	Uncommitted bool
	Unpushed    bool
}

// SelectCodespaces gives the codespaces that a delete takes: stopped
// (state Shutdown), with no uncommitted and no unpushed change, of one of
// the repositories when repositories are named, and last used more than
// OlderThan days ago.
func SelectCodespaces(codespaces []Codespace, repositories []string, f Filter, now time.Time) []Codespace {
	var out []Codespace
	for _, c := range codespaces {
		if c.State != "Shutdown" || c.Uncommitted || c.Unpushed || !f.OlderThanCutoff(c.LastUsed, now) {
			continue
		}
		if len(repositories) > 0 && !contains(repositories, c.Repository) {
			continue
		}
		out = append(out, c)
	}
	return out
}

// KeptCodespaces counts the stopped codespaces of the repositories, or of
// all repositories when none is named, that stay because of unsaved work.
func KeptCodespaces(codespaces []Codespace, repositories []string) int {
	n := 0
	for _, c := range codespaces {
		if len(repositories) > 0 && !contains(repositories, c.Repository) {
			continue
		}
		if c.State == "Shutdown" && (c.Uncommitted || c.Unpushed) {
			n++
		}
	}
	return n
}

// CodespaceLine is the line of one codespace after --delete-codespaces.
func CodespaceLine(c Codespace, dryRun bool) string {
	return fmt.Sprintf("%s: codespace %s, state %s, last used %s, %s",
		c.Repository, c.Name, c.State, format.Date(c.LastUsed), ActionText("delete", dryRun))
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
