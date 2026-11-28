package gh

import (
	"fmt"
	"strings"

	"github.com/rbe/gh-sweep/internal/format"
	"github.com/rbe/gh-sweep/internal/project"
)

// unknownWorkflow is the workflow of an artifact whose run GitHub does not
// give, because the run was deleted or has no name.
const unknownWorkflow = "-"

// AddWorkflows sets the workflow of each artifact of the repository. It
// reads the name of each workflow run once. A run that GitHub answers with
// 404, or with no name, gives the workflow "-". Any other error stops the
// read and is returned.
func AddWorkflows(p *project.Project) error {
	names := map[string]string{}
	for i, it := range p.Items {
		if it.Kind != "artifact" {
			continue
		}
		if it.Run == "" || it.Run == "0" {
			p.Items[i].Workflow = unknownWorkflow
			continue
		}
		name, ok := names[it.Run]
		if !ok {
			var err error
			if name, err = workflowName(*p, it.Run); err != nil {
				return err
			}
			names[it.Run] = name
		}
		p.Items[i].Workflow = name
	}
	return nil
}

// workflowName reads the name of one workflow run.
func workflowName(p project.Project, run string) (string, error) {
	args := append(apiArgs(p), fmt.Sprintf("repos/%s/%s/actions/runs/%s", p.Owner, p.Name, run), "--jq", ".name // \"\"")
	out, err := runGHFunc(args...)
	if err != nil {
		if strings.HasPrefix(err.Error(), "HTTP 404") {
			return unknownWorkflow, nil
		}
		return "", err
	}
	name := strings.TrimSpace(format.StripControls(out))
	if name == "" {
		return unknownWorkflow, nil
	}
	return name, nil
}
