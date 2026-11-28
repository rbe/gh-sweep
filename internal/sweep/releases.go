package sweep

import (
	"fmt"
	"time"

	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
)

// releaseFilters are the filters of the two release actions.
var releaseFilters = []string{"older-than", "keep"}

var releaseActions = []Action{
	{Flag: "--delete-releases", Title: "Delete releases and keep their tags", Verb: "delete", Noun: "releases",
		Filters: releaseFilters, NeedsFilter: true, TotalLine: project.SizeTotalLine("releases"), List: releaseEntries},
	{Flag: "--delete-release-assets", Title: "Delete the files of releases", Verb: "delete", Noun: "release assets",
		Filters: releaseFilters, NeedsFilter: true, TotalLine: project.SizeTotalLine("release assets"), List: assetEntries},
}

var accountActions = []Action{
	{Flag: "--delete-package-versions", Title: "Delete versions of a package", Verb: "delete", Noun: "package versions",
		NeedsFilter: true, Filters: []string{"package", "package-type", "untagged", "older-than", "keep"},
		Account: packageEntries, Check: checkPackageScope},
	{Flag: "--delete-codespaces", Title: "Delete your stopped codespaces", Verb: "delete", Noun: "codespaces",
		Filters: []string{"older-than"}, Account: codespaceEntries, Check: checkCodespaceScope},
}

// releaseEntries gives one entry for each selected release. The write
// deletes the release, and its tag stays.
func releaseEntries(p project.Project, f project.Filter, now time.Time) ([]Entry, error) {
	releases, err := gh.ListReleases(p)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for _, r := range project.SelectReleases(releases, f, now) {
		r := r
		entries = append(entries, counted(
			func(dry bool) string { return project.ReleaseLine(p.NWO, r, dry) }, r.Bytes(),
			func() error { return gh.DeleteRelease(p, r.ID) }))
	}
	return entries, nil
}

// assetEntries gives one entry for each asset of the selected releases.
// An immutable release keeps its assets, because GitHub refuses the delete.
func assetEntries(p project.Project, f project.Filter, now time.Time) ([]Entry, error) {
	releases, err := gh.ListReleases(p)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for _, r := range project.SelectReleases(releases, f, now) {
		if r.Immutable {
			continue
		}
		for _, a := range r.Assets {
			r, a := r, a
			entries = append(entries, counted(
				func(dry bool) string { return project.AssetLine(p.NWO, r, a, dry) }, a.Size,
				func() error { return gh.DeleteReleaseAsset(p, a.ID) }))
		}
	}
	return entries, nil
}

// checkPackageScope needs --package, a known --package-type, and --mine or
// --org. A package of another user cannot be deleted.
func checkPackageScope(s Scope, f project.Filter) error {
	if err := project.CheckPackage(f); err != nil {
		return err
	}
	if len(s.Repos) > 0 || s.User != "" || (!s.Mine && s.Org == "") {
		return fmt.Errorf("--delete-package-versions deletes packages of --mine or --org NAME. Pass one of them and no repository")
	}
	return nil
}

// packageEntries gives one entry for each selected version of the package
// of the logged-in user or of the organization.
func packageEntries(s Scope, f project.Filter, now time.Time) ([]Entry, error) {
	owner := s.Org
	if owner == "" {
		owner = "user"
	}
	versions, err := gh.ListPackageVersions(s.Org, f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", owner, err)
	}
	var entries []Entry
	for _, v := range project.SelectPackageVersions(versions, f, now) {
		v := v
		entries = append(entries, counted(
			func(dry bool) string { return project.PackageVersionLine(owner, f, v, dry) }, 0,
			func() error {
				if err := gh.DeletePackageVersion(s.Org, f, v.ID); err != nil {
					return fmt.Errorf("%s: %w", owner, err)
				}
				return nil
			}))
	}
	return entries, nil
}

// checkCodespaceScope needs --mine or repository arguments. --org and
// --user are an error, because the API lists only the codespaces of the
// logged-in user.
func checkCodespaceScope(s Scope, _ project.Filter) error {
	if s.Org != "" || s.User != "" || (!s.Mine && len(s.Repos) == 0) {
		return fmt.Errorf("--delete-codespaces deletes your own codespaces. Pass --mine, or name the repositories")
	}
	return nil
}

// codespaceEntries gives one entry for each stopped codespace with no
// unsaved work, in the named repositories or in all of them with --mine,
// and a line that counts the stopped codespaces that stay.
func codespaceEntries(s Scope, f project.Filter, now time.Time) ([]Entry, error) {
	codespaces, err := gh.ListCodespaces()
	if err != nil {
		return nil, err
	}
	repositories, err := parseNames(s.Repos)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, p := range repositories {
		names = append(names, p.NWO)
	}
	var entries []Entry
	for _, c := range project.SelectCodespaces(codespaces, names, f, now) {
		c := c
		entries = append(entries, counted(
			func(dry bool) string { return project.CodespaceLine(c, dry) }, 0,
			func() error {
				if err := gh.DeleteCodespace(c.Name); err != nil {
					return fmt.Errorf("%s: %w", c.Name, err)
				}
				return nil
			}))
	}
	if kept := project.KeptCodespaces(codespaces, names); kept > 0 {
		line := fmt.Sprintf("kept: %d stopped codespaces with uncommitted or unpushed changes", kept)
		entries = append(entries, Entry{Line: func(bool) string { return line }})
	}
	return entries, nil
}
