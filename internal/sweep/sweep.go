// Package sweep holds the actions that the write flags and the action
// screen of the checklist run: what each action reads, which entries it
// selects, and the write of each entry. The command line and the
// checklist use the same actions, so both show the same lines.
package sweep

import (
	"fmt"
	"strings"
	"time"

	"github.com/rbe/gh-sweep/internal/project"
)

// Scope is the target of an action: named repositories, or the
// repositories of --org, --user, or --mine.
type Scope struct {
	Repos []string
	Org   string
	User  string
	Mine  bool
}

// Named is true when the scope names repositories, an organization, a
// user, or the logged-in user.
func (s Scope) Named() bool {
	return len(s.Repos) > 0 || s.Org != "" || s.User != "" || s.Mine
}

// Entry is one line of an action: the text of the line, what it counts in
// the total, and the write. An entry with no write is a line to read, such
// as a usage line or a line that says what stays.
type Entry struct {
	Line  func(dryRun bool) string
	Total project.Total
	Apply func() error
}

// Action is one write flag or read-only flag. List reads one repository and
// gives its entries. Account, when set, replaces List and runs once. Check,
// when set, checks the scope in place of the rule that a write needs a
// named scope.
type Action struct {
	Flag        string
	Title       string
	Verb        string
	Noun        string
	Filters     []string
	NeedsFilter bool
	ReadOnly    bool
	TotalLine   func(project.Total, bool) string
	List        func(p project.Project, f project.Filter, now time.Time) ([]Entry, error)
	Account     func(s Scope, f project.Filter, now time.Time) ([]Entry, error)
	Check       func(s Scope, f project.Filter) error
}

// Total is the last line of the action.
func (a Action) Total(t project.Total, dryRun bool) string {
	if a.TotalLine != nil {
		return a.TotalLine(t, dryRun)
	}
	return project.TotalLine(a.Verb, a.Noun, t.Count, dryRun)
}

// Reads is true when the action reads the filter.
func (a Action) Reads(filter string) bool {
	return contains(a.Filters, filter)
}

// PackageFilters name the package. They select no version, so they do not
// count as a filter.
var PackageFilters = []string{"package", "package-type"}

// CheckFilters refuses a filter that the action does not read, an action
// that needs a filter and has none, a run status that GitHub does not
// accept, and a scope that the action does not take.
func (a Action) CheckFilters(given []string, f project.Filter, s Scope) error {
	filtered := false
	for _, name := range given {
		if !a.Reads(name) {
			return fmt.Errorf("%s does not read --%s", a.Flag, name)
		}
		filtered = filtered || !contains(PackageFilters, name)
	}
	if a.NeedsFilter && !filtered {
		var selecting []string
		for _, name := range a.Filters {
			if !contains(PackageFilters, name) {
				selecting = append(selecting, name)
			}
		}
		return fmt.Errorf("%s needs at least one of --%s", a.Flag, strings.Join(selecting, ", --"))
	}
	switch {
	case a.Check != nil:
		if err := a.Check(s, f); err != nil {
			return err
		}
	case !a.ReadOnly && !s.Named():
		return fmt.Errorf("to use %s, name the repositories, or pass --mine, --org, or --user", a.Flag)
	}
	return project.CheckRunStatus(f.Status)
}

// Apply writes each entry in order, unless dryRun is set, and gives the
// lines and the total of the entries done. It stops at the first error.
func Apply(entries []Entry, dryRun bool) ([]string, project.Total, error) {
	var lines []string
	var t project.Total
	for _, e := range entries {
		if !dryRun && e.Apply != nil {
			if err := e.Apply(); err != nil {
				return lines, t, err
			}
		}
		lines = append(lines, e.Line(dryRun))
		t = t.Add(e.Total)
	}
	return lines, t, nil
}

// All are the actions, in the order of the help text.
func All() []Action {
	var all []Action
	for _, group := range [][]Action{runActions, cacheActions, releaseActions, accountActions,
		{billingAction, branchAction}, deploymentActions, staleActions} {
		all = append(all, group...)
	}
	return all
}

// ByFlag gives the action of the flag, such as --delete-runs.
func ByFlag(flag string) (Action, bool) {
	for _, a := range All() {
		if a.Flag == flag {
			return a, true
		}
	}
	return Action{}, false
}

// parseNames parses each name as a repository.
func parseNames(names []string) ([]project.Project, error) {
	var out []project.Project
	for _, name := range names {
		p, err := project.Parse(name)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// counted is an entry that counts one and the bytes.
func counted(line func(bool) string, bytes int64, apply func() error) Entry {
	return Entry{Line: line, Total: project.Total{Count: 1, Bytes: bytes}, Apply: apply}
}
