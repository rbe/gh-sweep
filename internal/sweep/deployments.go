package sweep

import (
	"time"

	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
)

var deploymentActions = []Action{
	{Flag: "--delete-deployments", Title: "Delete old deployments", Verb: "delete", Noun: "deployments", NeedsFilter: true,
		Filters: []string{"older-than", "keep"}, List: deploymentEntries},
	{Flag: "--delete-unused-environments", Title: "Delete unused environments", Verb: "delete", Noun: "environments",
		Filters: []string{"older-than"}, List: environmentEntries},
}

// deploymentEntries gives one entry for each selected deployment.
func deploymentEntries(p project.Project, f project.Filter, now time.Time) ([]Entry, error) {
	deployments, err := gh.ListDeployments(p)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for _, d := range project.SelectDeployments(deployments, f, now) {
		d := d
		entries = append(entries, counted(
			func(dry bool) string { return project.DeploymentLine(p.NWO, d, dry) }, 0,
			func() error { return gh.DeleteDeployment(p, d.ID) }))
	}
	return entries, nil
}

// environmentEntries gives one entry for each environment with no
// deployment, or with no deployment in the last OlderThan days.
func environmentEntries(p project.Project, f project.Filter, now time.Time) ([]Entry, error) {
	environments, err := gh.ListEnvironments(p)
	if err != nil {
		return nil, err
	}
	deployments, err := gh.ListDeployments(p)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for _, e := range project.SelectUnusedEnvironments(environments, deployments, f, now) {
		e := e
		entries = append(entries, counted(
			func(dry bool) string { return project.EnvironmentLine(p.NWO, e, dry) }, 0,
			func() error { return gh.DeleteEnvironment(p, e.Name) }))
	}
	return entries, nil
}
