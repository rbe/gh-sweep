package main

import (
	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
)

// deploymentJobs are --delete-deployments and --delete-unused-environments.
var deploymentJobs = []job{
	{flag: "--delete-deployments", verb: "delete", noun: "deployments", needsFilter: true,
		filters: []string{"older-than", "keep"}, each: deleteDeployments},
	{flag: "--delete-unused-environments", verb: "delete", noun: "environments",
		filters: []string{"older-than"}, each: deleteUnusedEnvironments},
}

// deleteDeployments deletes the selected deployments of one repository.
// It stops at the first error.
func deleteDeployments(p project.Project, f project.Filter, dryRun bool) ([]string, project.Total, error) {
	deployments, err := gh.ListDeployments(p)
	if err != nil {
		return nil, project.Total{}, err
	}
	var lines []string
	for _, d := range project.SelectDeployments(deployments, f, now()) {
		if !dryRun {
			if err := gh.DeleteDeployment(p, d.ID); err != nil {
				return lines, project.Total{Count: len(lines)}, err
			}
		}
		lines = append(lines, project.DeploymentLine(p.NWO, d, dryRun))
	}
	return lines, project.Total{Count: len(lines)}, nil
}

// deleteUnusedEnvironments deletes the environments of one repository with
// no deployment, or with no deployment in the last OlderThan days. It
// stops at the first error.
func deleteUnusedEnvironments(p project.Project, f project.Filter, dryRun bool) ([]string, project.Total, error) {
	environments, err := gh.ListEnvironments(p)
	if err != nil {
		return nil, project.Total{}, err
	}
	deployments, err := gh.ListDeployments(p)
	if err != nil {
		return nil, project.Total{}, err
	}
	var lines []string
	for _, e := range project.SelectUnusedEnvironments(environments, deployments, f, now()) {
		if !dryRun {
			if err := gh.DeleteEnvironment(p, e.Name); err != nil {
				return lines, project.Total{Count: len(lines)}, err
			}
		}
		lines = append(lines, project.EnvironmentLine(p.NWO, e, dryRun))
	}
	return lines, project.Total{Count: len(lines)}, nil
}
