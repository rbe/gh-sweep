package gh

import (
	"fmt"
	"strings"

	"github.com/rbe/gh-sweep/internal/project"
)

const retentionPath = "repos/%s/%s/actions/permissions/artifact-and-log-retention"
const workflowPath = "repos/%s/%s/actions/permissions/workflow"

// apiArgs starts a gh api call for the host of the repository.
func apiArgs(p project.Project) []string {
	args := []string{"api"}
	if p.APIHost != "" {
		args = append(args, "--hostname", p.APIHost)
	}
	return append(args, "-H", "Accept: application/vnd.github+json")
}

// GetRetention reads the artifact and log retention of the repository.
func GetRetention(p project.Project) (project.Retention, error) {
	args := append(apiArgs(p), fmt.Sprintf(retentionPath, p.Owner, p.Name),
		"--jq", `"\(.days) \(.maximum_allowed_days)"`)
	out, err := runGHFunc(args...)
	if err != nil {
		return project.Retention{}, err
	}
	fields := strings.Fields(out)
	if len(fields) != 2 {
		return project.Retention{}, fmt.Errorf("gh gave no retention: %q", strings.TrimSpace(out))
	}
	return project.Retention{Days: int(atoi64(fields[0])), MaxDays: int(atoi64(fields[1]))}, nil
}

// SetRetention sets the artifact and log retention of the repository.
func SetRetention(p project.Project, days int) error {
	args := append(apiArgs(p), "-X", "PUT", fmt.Sprintf(retentionPath, p.Owner, p.Name),
		"-F", fmt.Sprintf("days=%d", days))
	return retryGH(args...)
}

// GetWorkflowPermissions reads the GITHUB_TOKEN setting of the repository.
func GetWorkflowPermissions(p project.Project) (project.WorkflowPermissions, error) {
	args := append(apiArgs(p), fmt.Sprintf(workflowPath, p.Owner, p.Name),
		"--jq", `"\(.default_workflow_permissions) \(.can_approve_pull_request_reviews)"`)
	out, err := runGHFunc(args...)
	if err != nil {
		return project.WorkflowPermissions{}, err
	}
	fields := strings.Fields(out)
	if len(fields) != 2 || (fields[1] != "true" && fields[1] != "false") {
		return project.WorkflowPermissions{}, fmt.Errorf("gh gave no workflow permissions: %q", strings.TrimSpace(out))
	}
	return project.WorkflowPermissions{Default: fields[0], CanApprovePRs: fields[1] == "true"}, nil
}

// SetWorkflowPermissions writes both values of the GITHUB_TOKEN setting, so
// that the default permission stays as it was read.
func SetWorkflowPermissions(p project.Project, w project.WorkflowPermissions) error {
	args := append(apiArgs(p), "-X", "PUT", fmt.Sprintf(workflowPath, p.Owner, p.Name),
		"-f", "default_workflow_permissions="+w.Default,
		"-F", fmt.Sprintf("can_approve_pull_request_reviews=%t", w.CanApprovePRs))
	return retryGH(args...)
}
