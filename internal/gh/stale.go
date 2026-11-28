package gh

import (
	"fmt"

	"github.com/rbe/gh-sweep/internal/format"
	"github.com/rbe/gh-sweep/internal/project"
)

const stateFields = "nameWithOwner,pushedAt,diskUsage,isArchived"

const stateJQ = `[
    .nameWithOwner,
    (.pushedAt // ""),
    ((.diskUsage // 0) | tostring),
    (if .isArchived then "1" else "0" end)
  ]
| join("\u001f")`

// ListRepositoryStates reads the last push, the disk usage, and the archive
// state of the first ListCap repositories of the owner, or of the
// logged-in user when who is empty.
func ListRepositoryStates(who string) ([]project.RepositoryState, error) {
	args := []string{"repo", "list"}
	if who != "" {
		args = append(args, who)
	}
	out, err := runGHFunc(append(args, "--limit", fmt.Sprint(ListCap), "--json", stateFields, "--jq", ".[] | "+stateJQ)...)
	if err != nil {
		return nil, err
	}
	return parseStates(out), nil
}

// RepositoryStateOf reads the state of one repository.
func RepositoryStateOf(p project.Project) (project.RepositoryState, error) {
	out, err := runGHFunc("repo", "view", p.NWO, "--json", stateFields, "--jq", stateJQ)
	if err != nil {
		return project.RepositoryState{}, err
	}
	states := parseStates(out)
	if len(states) != 1 {
		return project.RepositoryState{}, fmt.Errorf("gh gave no repository state")
	}
	states[0].NWO = p.NWO
	return states[0], nil
}

// parseStates reads the lines of stateJQ. GitHub gives the disk usage in
// kilobytes.
func parseStates(raw string) []project.RepositoryState {
	var states []project.RepositoryState
	for _, f := range fieldLines(raw, 4) {
		states = append(states, project.RepositoryState{
			NWO: f[0], Pushed: format.ParseGitHubTime(f[1]), Disk: atoi64(f[2]) * 1024, Archived: f[3] == "1",
		})
	}
	return states
}

// Archive archives the repository. GitHub then makes it read-only.
func Archive(p project.Project) error {
	return retryGH(append(apiArgs(p), "-X", "PATCH", fmt.Sprintf("repos/%s/%s", p.Owner, p.Name), "-F", "archived=true")...)
}
