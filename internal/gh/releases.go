package gh

import (
	"fmt"
	"net/url"

	"github.com/rbe/gh-sweep/internal/format"
	"github.com/rbe/gh-sweep/internal/project"
)

// releaseJQ writes one R line for each release and one A line for each
// asset after its release.
const releaseJQ = `.[]?
| ([
    "R",
    (.id | tostring),
    ((.name // "") | gsub("[\t\r\n\u001f]"; " ")),
    ((.tag_name // "") | gsub("[\t\r\n\u001f]"; " ")),
    (if .draft then "1" else "0" end),
    (if .immutable then "1" else "0" end),
    (.created_at // "")
  ] | join("\u001f")),
  (.assets[]? | [
    "A",
    (.id | tostring),
    ((.name // "") | gsub("[\t\r\n\u001f]"; " ")),
    ((.size // 0) | tostring)
  ] | join("\u001f"))`

// ListReleases reads the releases of the repository and their assets.
func ListReleases(p project.Project) ([]project.Release, error) {
	args := append(apiArgs(p), "--paginate",
		fmt.Sprintf("repos/%s/%s/releases?per_page=100", p.Owner, p.Name), "--jq", releaseJQ)
	out, err := runGHFunc(args...)
	if err != nil {
		return nil, err
	}
	return ParseReleases(out), nil
}

// ParseReleases reads the lines that releaseJQ writes.
func ParseReleases(raw string) []project.Release {
	var releases []project.Release
	for _, f := range fieldLines(raw, 4) {
		switch {
		case f[0] == "R" && len(f) >= 7:
			releases = append(releases, project.Release{
				ID: f[1], Name: f[2], Tag: f[3], Draft: f[4] == "1", Immutable: f[5] == "1",
				Created: format.ParseGitHubTime(f[6]),
			})
		case f[0] == "A" && len(releases) > 0:
			last := &releases[len(releases)-1]
			last.Assets = append(last.Assets, project.Asset{ID: f[1], Name: f[2], Size: atoi64(f[3])})
		}
	}
	return releases
}

// DeleteRelease deletes the release. Its tag stays.
func DeleteRelease(p project.Project, id string) error {
	return retryGH(append(apiArgs(p), "-X", "DELETE", fmt.Sprintf("repos/%s/%s/releases/%s", p.Owner, p.Name, id))...)
}

// DeleteReleaseAsset deletes one asset of a release.
func DeleteReleaseAsset(p project.Project, id string) error {
	return retryGH(append(apiArgs(p), "-X", "DELETE", fmt.Sprintf("repos/%s/%s/releases/assets/%s", p.Owner, p.Name, id))...)
}

// packagePath is the path of a package of the logged-in user, or of the
// organization when org is not empty.
func packagePath(org string, f project.Filter) string {
	if org != "" {
		return fmt.Sprintf("orgs/%s/packages/%s/%s", org, f.PackageType, url.PathEscape(f.Package))
	}
	return fmt.Sprintf("user/packages/%s/%s", f.PackageType, url.PathEscape(f.Package))
}

const packageVersionJQ = `.[]?
| [
    (.id | tostring),
    ((.name // "") | gsub("[\t\r\n\u001f]"; " ")),
    (((.metadata // {}) | (.container // .docker // {}) | (.tags // .tag // [])) | join(",")),
    (.created_at // "")
  ]
| join("\u001f")`

// ListPackageVersions reads the versions of the package.
func ListPackageVersions(org string, f project.Filter) ([]project.PackageVersion, error) {
	out, err := runGHFunc("api", "-H", "Accept: application/vnd.github+json", "--paginate",
		packagePath(org, f)+"/versions?per_page=100", "--jq", packageVersionJQ)
	if err != nil {
		return nil, err
	}
	return ParsePackageVersions(out), nil
}

// ParsePackageVersions reads the lines that packageVersionJQ writes.
func ParsePackageVersions(raw string) []project.PackageVersion {
	var versions []project.PackageVersion
	for _, f := range fieldLines(raw, 4) {
		v := project.PackageVersion{ID: f[0], Name: f[1], Created: format.ParseGitHubTime(f[3])}
		for _, tag := range splitNonEmpty(f[2], ",") {
			v.Tags = append(v.Tags, tag)
		}
		versions = append(versions, v)
	}
	return versions
}

// DeletePackageVersion deletes one version of the package.
func DeletePackageVersion(org string, f project.Filter, id string) error {
	return retryGH("api", "-H", "Accept: application/vnd.github+json", "-X", "DELETE", packagePath(org, f)+"/versions/"+id)
}

const codespaceJQ = `.codespaces[]?
| [
    (.name // ""),
    ((.repository // {}) | .full_name // ""),
    (.state // ""),
    (.last_used_at // ""),
    (if (.git_status // {}) | .has_uncommitted_changes then "1" else "0" end),
    (if (.git_status // {}) | .has_unpushed_changes then "1" else "0" end)
  ]
| join("\u001f")`

// ListCodespaces reads the codespaces of the logged-in user.
func ListCodespaces() ([]project.Codespace, error) {
	out, err := runGHFunc("api", "-H", "Accept: application/vnd.github+json", "--paginate",
		"user/codespaces?per_page=100", "--jq", codespaceJQ)
	if err != nil {
		return nil, err
	}
	var codespaces []project.Codespace
	for _, f := range fieldLines(out, 6) {
		codespaces = append(codespaces, project.Codespace{
			Name: f[0], Repository: f[1], State: f[2], LastUsed: format.ParseGitHubTime(f[3]),
			Uncommitted: f[4] == "1", Unpushed: f[5] == "1",
		})
	}
	return codespaces, nil
}

// DeleteCodespace deletes one codespace of the logged-in user.
func DeleteCodespace(name string) error {
	return retryGH("api", "-H", "Accept: application/vnd.github+json", "-X", "DELETE", "user/codespaces/"+url.PathEscape(name))
}
