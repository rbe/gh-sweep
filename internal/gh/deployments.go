package gh

import (
	"fmt"
	"net/url"

	"github.com/rbe/gh-sweep/internal/format"
	"github.com/rbe/gh-sweep/internal/project"
)

const deploymentJQ = `.[]?
| [
    (.id | tostring),
    ((.environment // "") | gsub("[\t\r\n\u001f]"; " ")),
    ((.ref // "") | gsub("[\t\r\n\u001f]"; " ")),
    (.created_at // "")
  ]
| join("\u001f")`

// ListDeployments reads the deployments of the repository.
func ListDeployments(p project.Project) ([]project.Deployment, error) {
	out, err := runGHFunc(append(apiArgs(p), "--paginate",
		fmt.Sprintf("repos/%s/%s/deployments?per_page=100", p.Owner, p.Name), "--jq", deploymentJQ)...)
	if err != nil {
		return nil, err
	}
	var deployments []project.Deployment
	for _, f := range fieldLines(out, 4) {
		deployments = append(deployments, project.Deployment{
			ID: f[0], Environment: f[1], Ref: f[2], Created: format.ParseGitHubTime(f[3]),
		})
	}
	return deployments, nil
}

// DeleteDeployment sets the deployment inactive, because GitHub deletes
// only an inactive deployment when a repository has more than one, and
// then deletes it.
func DeleteDeployment(p project.Project, id string) error {
	path := fmt.Sprintf("repos/%s/%s/deployments/%s", p.Owner, p.Name, id)
	if err := retryGH(append(apiArgs(p), "-X", "POST", path+"/statuses", "-f", "state=inactive")...); err != nil {
		return err
	}
	return retryGH(append(apiArgs(p), "-X", "DELETE", path)...)
}

// ListEnvironments reads the names of the environments of the repository.
func ListEnvironments(p project.Project) ([]string, error) {
	out, err := runGHFunc(append(apiArgs(p), "--paginate",
		fmt.Sprintf("repos/%s/%s/environments?per_page=100", p.Owner, p.Name),
		"--jq", `.environments[]? | (.name // "") | gsub("[\t\r\n\u001f]"; " ")`)...)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, f := range fieldLines(out, 1) {
		names = append(names, f[0])
	}
	return names, nil
}

// DeleteEnvironment deletes the environment with its secrets, variables,
// and protection rules.
func DeleteEnvironment(p project.Project, name string) error {
	return retryGH(append(apiArgs(p), "-X", "DELETE",
		fmt.Sprintf("repos/%s/%s/environments/%s", p.Owner, p.Name, url.PathEscape(name)))...)
}
