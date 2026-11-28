package gh

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/rbe/gh-sweep/internal/project"
)

const branchJQ = `.[]?
| [
    (.name // ""),
    ((.commit // {}) | .sha // ""),
    (if .protected then "1" else "0" end)
  ]
| join("\u001f")`

const pullJQ = `.[]?
| [
    (.number | tostring),
    (.state // ""),
    (if .merged_at then "1" else "0" end),
    ((.head // {}) | .ref // ""),
    ((.head // {}) | .sha // ""),
    (((.head // {}) | .repo // {}) | .full_name // ""),
    ((.base // {}) | .ref // "")
  ]
| join("\u001f")`

// DefaultBranch reads the default branch of the repository.
func DefaultBranch(p project.Project) (string, error) {
	out, err := runGHFunc(append(apiArgs(p), fmt.Sprintf("repos/%s/%s", p.Owner, p.Name), "--jq", ".default_branch")...)
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(out)
	if name == "" {
		return "", fmt.Errorf("gh gave no default branch")
	}
	return name, nil
}

// ListBranches reads the branches of the repository.
func ListBranches(p project.Project) ([]project.Branch, error) {
	out, err := runGHFunc(append(apiArgs(p), "--paginate",
		fmt.Sprintf("repos/%s/%s/branches?per_page=100", p.Owner, p.Name), "--jq", branchJQ)...)
	if err != nil {
		return nil, err
	}
	var branches []project.Branch
	for _, f := range fieldLines(out, 3) {
		branches = append(branches, project.Branch{Name: f[0], SHA: f[1], Protected: f[2] == "1"})
	}
	return branches, nil
}

// ListPullRequests reads the open and the closed pull requests.
func ListPullRequests(p project.Project) ([]project.PullRequest, error) {
	out, err := runGHFunc(append(apiArgs(p), "--paginate",
		fmt.Sprintf("repos/%s/%s/pulls?state=all&per_page=100", p.Owner, p.Name), "--jq", pullJQ)...)
	if err != nil {
		return nil, err
	}
	var pulls []project.PullRequest
	for _, f := range fieldLines(out, 7) {
		pulls = append(pulls, project.PullRequest{
			Number: f[0], State: f[1], Merged: f[2] == "1", HeadRef: f[3], HeadSHA: f[4], HeadRepo: f[5], BaseRef: f[6],
		})
	}
	return pulls, nil
}

// DeleteBranch deletes the branch. Each part of the name between slashes
// is escaped for the path.
func DeleteBranch(p project.Project, name string) error {
	parts := strings.Split(name, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return retryGH(append(apiArgs(p), "-X", "DELETE",
		fmt.Sprintf("repos/%s/%s/git/refs/heads/%s", p.Owner, p.Name, strings.Join(parts, "/")))...)
}
