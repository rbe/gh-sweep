package gh

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/rbe/gh-sweep/internal/format"
	"github.com/rbe/gh-sweep/internal/project"
)

const runJQ = `.workflow_runs[]?
| [
    (.id | tostring),
    ((.name // "") | gsub("[\t\r\n\u001f]"; " ")),
    ((.head_branch // "") | gsub("[\t\r\n\u001f]"; " ")),
    (.status // ""),
    (.conclusion // ""),
    (.created_at // "")
  ]
| join("\u001f")`

// ListRuns reads the workflow runs of the repository. A branch or a status
// that is not empty goes to GitHub as a filter. GitHub then gives at most
// 1,000 runs.
func ListRuns(p project.Project, branch, status string) ([]project.Run, error) {
	q := url.Values{"per_page": {"100"}}
	if branch != "" {
		q.Set("branch", branch)
	}
	if status != "" {
		q.Set("status", status)
	}
	args := append(apiArgs(p), "--paginate",
		fmt.Sprintf("repos/%s/%s/actions/runs?%s", p.Owner, p.Name, q.Encode()), "--jq", runJQ)
	out, err := runGHFunc(args...)
	if err != nil {
		return nil, err
	}
	return ParseRuns(out), nil
}

// ParseRuns reads the lines that runJQ writes.
func ParseRuns(raw string) []project.Run {
	var runs []project.Run
	for _, f := range fieldLines(raw, 6) {
		runs = append(runs, project.Run{
			ID:         f[0],
			Name:       f[1],
			Branch:     f[2],
			Status:     f[3],
			Conclusion: f[4],
			Created:    format.ParseGitHubTime(f[5]),
		})
	}
	return runs
}

// DeleteRun deletes the workflow run and its logs.
func DeleteRun(p project.Project, id string) error {
	return retryGH(append(apiArgs(p), "-X", "DELETE", fmt.Sprintf("repos/%s/%s/actions/runs/%s", p.Owner, p.Name, id))...)
}

// DeleteRunLogs deletes the logs of the workflow run and keeps the run.
func DeleteRunLogs(p project.Project, id string) error {
	return retryGH(append(apiArgs(p), "-X", "DELETE", fmt.Sprintf("repos/%s/%s/actions/runs/%s/logs", p.Owner, p.Name, id))...)
}

// fieldLines splits the output of a jq program into lines of at least n
// fields, separated by U+001F. It drops a line with fewer fields or an
// empty first field, and strips control characters from each field.
func fieldLines(raw string, n int) [][]string {
	var out [][]string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		f := strings.Split(line, "\x1f")
		if len(f) < n || strings.TrimSpace(f[0]) == "" {
			continue
		}
		for i := range f {
			f[i] = format.StripControls(f[i])
		}
		out = append(out, f)
	}
	return out
}

// splitNonEmpty splits s at sep and drops empty parts.
func splitNonEmpty(s, sep string) []string {
	var out []string
	for _, part := range strings.Split(s, sep) {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
