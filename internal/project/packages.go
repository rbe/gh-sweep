package project

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rbe/gh-sweep/internal/format"
)

// PackageTypes are the package types of the GitHub Packages API.
var PackageTypes = []string{"container", "npm", "maven", "rubygems", "docker", "nuget"}

// PackageVersion is one version of a package. Name is the version, or the
// digest of a container image.
type PackageVersion struct {
	ID      string
	Name    string
	Tags    []string
	Created time.Time
}

// SelectPackageVersions gives the versions that a delete takes. The newest
// version always stays, and so do the newest Keep. With Untagged only
// versions without a tag are taken.
func SelectPackageVersions(versions []PackageVersion, f Filter, now time.Time) []PackageVersion {
	sorted := append([]PackageVersion{}, versions...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Created.After(sorted[j].Created) })
	keep := f.Keep
	if keep < 1 {
		keep = 1
	}
	var out []PackageVersion
	for i, v := range sorted {
		if i < keep || (f.Untagged && len(v.Tags) > 0) || !f.OlderThanCutoff(v.Created, now) {
			continue
		}
		out = append(out, v)
	}
	return out
}

// PackageVersionLine is the line of one version after
// --delete-package-versions.
func PackageVersionLine(owner string, f Filter, v PackageVersion, dryRun bool) string {
	tags := "-"
	if len(v.Tags) > 0 {
		tags = strings.Join(v.Tags, ",")
	}
	return fmt.Sprintf("%s: %s %s version %s %s, tags %s, created %s, %s",
		owner, f.PackageType, f.Package, v.ID, v.Name, tags, format.Date(v.Created), ActionText("delete", dryRun))
}

// CheckPackage refuses a package flag with no name and an unknown type.
func CheckPackage(f Filter) error {
	if f.Package == "" {
		return fmt.Errorf("--delete-package-versions needs --package NAME")
	}
	for _, t := range PackageTypes {
		if t == f.PackageType {
			return nil
		}
	}
	return fmt.Errorf("--package-type %s is not one of %s", f.PackageType, strings.Join(PackageTypes, ", "))
}
