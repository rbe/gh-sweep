package project

import (
	"fmt"
	"sort"
	"time"

	"github.com/rbe/gh-sweep/internal/format"
)

// Deployment is one deployment of a repository.
type Deployment struct {
	ID          string
	Environment string
	Ref         string
	Created     time.Time
}

// SelectDeployments gives the deployments that --delete-deployments takes.
// In each environment the newest Keep deployments stay, and at least the
// newest one, because it can be the active deployment. Only deployments
// created more than OlderThan days ago are taken.
func SelectDeployments(deployments []Deployment, f Filter, now time.Time) []Deployment {
	sorted := append([]Deployment{}, deployments...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Created.After(sorted[j].Created) })
	keep := f.Keep
	if keep < 1 {
		keep = 1
	}
	seen := map[string]int{}
	var out []Deployment
	for _, d := range sorted {
		seen[d.Environment]++
		if seen[d.Environment] <= keep || !f.OlderThanCutoff(d.Created, now) {
			continue
		}
		out = append(out, d)
	}
	return out
}

// DeploymentLine is the line of one deployment after --delete-deployments.
func DeploymentLine(nwo string, d Deployment, dryRun bool) string {
	return fmt.Sprintf("%s: deployment %s, environment %s, ref %s, created %s, %s",
		nwo, d.ID, dash(d.Environment), dash(d.Ref), format.Date(d.Created), ActionText("delete", dryRun))
}

// UnusedEnvironment is an environment that --delete-unused-environments
// takes, with the number of its deployments and the time of the newest.
type UnusedEnvironment struct {
	Name        string
	Deployments int
	Newest      time.Time
}

// SelectUnusedEnvironments gives the environments with no deployment.
// With OlderThan it also gives the environments whose newest deployment
// was created more than OlderThan days ago.
func SelectUnusedEnvironments(environments []string, deployments []Deployment, f Filter, now time.Time) []UnusedEnvironment {
	count := map[string]int{}
	newest := map[string]time.Time{}
	for _, d := range deployments {
		count[d.Environment]++
		if d.Created.After(newest[d.Environment]) {
			newest[d.Environment] = d.Created
		}
	}
	var out []UnusedEnvironment
	for _, name := range environments {
		n := count[name]
		if n > 0 && (f.OlderThan == 0 || !f.OlderThanCutoff(newest[name], now)) {
			continue
		}
		out = append(out, UnusedEnvironment{Name: name, Deployments: n, Newest: newest[name]})
	}
	return out
}

// EnvironmentLine is the line of one environment after
// --delete-unused-environments.
func EnvironmentLine(nwo string, e UnusedEnvironment, dryRun bool) string {
	return fmt.Sprintf("%s: environment %s, deployments %d, newest deployment %s, %s",
		nwo, e.Name, e.Deployments, format.Date(e.Newest), ActionText("delete", dryRun))
}
