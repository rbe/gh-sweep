package main

import (
	"fmt"

	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
	"github.com/rbe/gh-sweep/internal/ui"
)

// releaseFilters are the filter flags of the release flags.
var releaseFilters = []string{"older-than", "keep"}

// accountJobs are the flags of releases, packages, and codespaces.
var accountJobs = []job{
	{flag: "--delete-releases", verb: "delete", noun: "releases", filters: releaseFilters, needsFilter: true,
		total: project.SizeTotalLine("releases"), each: deleteReleases},
	{flag: "--delete-release-assets", verb: "delete", noun: "release assets", filters: releaseFilters, needsFilter: true,
		total: project.SizeTotalLine("release assets"), each: deleteReleaseAssets},
	{flag: "--delete-package-versions", verb: "delete", noun: "package versions", needsFilter: true,
		filters: []string{"package", "package-type", "untagged", "older-than", "keep"},
		account: deletePackageVersions, target: checkPackageTarget},
	{flag: "--delete-codespaces", verb: "delete", noun: "codespaces", filters: []string{"older-than"},
		account: deleteCodespaces, target: checkCodespaceTarget},
}

// deleteReleases deletes the selected releases of one repository. The
// tags stay. It stops at the first error.
func deleteReleases(p project.Project, f project.Filter, dryRun bool) ([]string, project.Total, error) {
	releases, err := gh.ListReleases(p)
	if err != nil {
		return nil, project.Total{}, err
	}
	var lines []string
	var t project.Total
	for _, r := range project.SelectReleases(releases, f, now()) {
		if !dryRun {
			if err := gh.DeleteRelease(p, r.ID); err != nil {
				return lines, t, err
			}
		}
		lines = append(lines, project.ReleaseLine(p.NWO, r, dryRun))
		t = t.Add(project.Total{Count: 1, Bytes: r.Bytes()})
	}
	return lines, t, nil
}

// deleteReleaseAssets deletes the assets of the selected releases of one
// repository and keeps the releases. An immutable release keeps its
// assets, because GitHub refuses the delete. It stops at the first error.
func deleteReleaseAssets(p project.Project, f project.Filter, dryRun bool) ([]string, project.Total, error) {
	releases, err := gh.ListReleases(p)
	if err != nil {
		return nil, project.Total{}, err
	}
	var lines []string
	var t project.Total
	for _, r := range project.SelectReleases(releases, f, now()) {
		if r.Immutable {
			continue
		}
		for _, a := range r.Assets {
			if !dryRun {
				if err := gh.DeleteReleaseAsset(p, a.ID); err != nil {
					return lines, t, err
				}
			}
			lines = append(lines, project.AssetLine(p.NWO, r, a, dryRun))
			t = t.Add(project.Total{Count: 1, Bytes: a.Size})
		}
	}
	return lines, t, nil
}

// checkPackageTarget needs --package, a known --package-type, and --mine or
// --org. A package of another user cannot be deleted.
func checkPackageTarget(o ui.Options, f project.Filter) error {
	if err := project.CheckPackage(f); err != nil {
		return err
	}
	if len(o.Repos) > 0 || o.User != "" || (!o.Mine && o.Org == "") {
		return fmt.Errorf("--delete-package-versions deletes packages of --mine or --org NAME. Pass one of them and no repository")
	}
	return nil
}

// deletePackageVersions deletes the selected versions of one package of
// the logged-in user or of the organization.
func deletePackageVersions(o ui.Options, f project.Filter, dryRun bool) ([]string, project.Total, error) {
	owner := o.Org
	if owner == "" {
		owner = "user"
	}
	versions, err := gh.ListPackageVersions(o.Org, f)
	if err != nil {
		return nil, project.Total{}, fmt.Errorf("%s: %w", owner, err)
	}
	var lines []string
	for _, v := range project.SelectPackageVersions(versions, f, now()) {
		if !dryRun {
			if err := gh.DeletePackageVersion(o.Org, f, v.ID); err != nil {
				return lines, project.Total{Count: len(lines)}, fmt.Errorf("%s: %w", owner, err)
			}
		}
		lines = append(lines, project.PackageVersionLine(owner, f, v, dryRun))
	}
	return lines, project.Total{Count: len(lines)}, nil
}

// checkCodespaceTarget needs --mine or repository arguments. --org and
// --user are an error, because the API lists only the codespaces of the
// logged-in user.
func checkCodespaceTarget(o ui.Options, _ project.Filter) error {
	if o.Org != "" || o.User != "" || (!o.Mine && len(o.Repos) == 0) {
		return fmt.Errorf("--delete-codespaces deletes your own codespaces. Pass --mine, or name the repositories")
	}
	return nil
}

// deleteCodespaces deletes the stopped codespaces of the logged-in user
// that have no unsaved work, in the named repositories or in all of them
// with --mine.
func deleteCodespaces(o ui.Options, f project.Filter, dryRun bool) ([]string, project.Total, error) {
	codespaces, err := gh.ListCodespaces()
	if err != nil {
		return nil, project.Total{}, err
	}
	repositories, err := parseRepositories(o.Repos)
	if err != nil {
		return nil, project.Total{}, err
	}
	var names []string
	for _, p := range repositories {
		names = append(names, p.NWO)
	}
	var lines []string
	for _, c := range project.SelectCodespaces(codespaces, names, f, now()) {
		if !dryRun {
			if err := gh.DeleteCodespace(c.Name); err != nil {
				return lines, project.Total{Count: len(lines)}, fmt.Errorf("%s: %w", c.Name, err)
			}
		}
		lines = append(lines, project.CodespaceLine(c, dryRun))
	}
	t := project.Total{Count: len(lines)}
	if kept := project.KeptCodespaces(codespaces, names); kept > 0 {
		lines = append(lines, fmt.Sprintf("kept: %d stopped codespaces with uncommitted or unpushed changes", kept))
	}
	return lines, t, nil
}
