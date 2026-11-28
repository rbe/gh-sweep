package cli

import (
	"fmt"
	"io"

	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
	"github.com/rbe/gh-sweep/internal/ui"
)

// report is one plain-text report: its name in a note, and the function
// that turns the loaded inventories into its lines.
type report struct {
	name  string
	lines func([]project.Project) []string
}

// printReport loads the inventories and writes the lines of the report to
// stdout. A repository that cannot be loaded gets a skipped line before the
// total line, the other repositories still get their lines, and the result
// is an error that counts the skipped repositories.
func printReport(o ui.Options, r *report, stdout, stderr io.Writer) error {
	if err := gh.Auth(); err != nil {
		return err
	}
	names, note, err := reportNames(o, r.name)
	if err != nil {
		return err
	}
	repositories, err := parseRepositories(names)
	if err != nil {
		return err
	}
	inventories, skipped := loadInventories(repositories, o.WantArt, o.WantCache)
	for _, line := range withSkipped(r.lines(inventories), skipped) {
		fmt.Fprintln(stdout, line)
	}
	if note != "" {
		fmt.Fprintf(stderr, "gh-sweep: %s\n", note)
	}
	if len(skipped) > 0 {
		return fmt.Errorf("the %s skipped %d of %d repositories", r.name, len(skipped), len(repositories))
	}
	return nil
}

// withSkipped puts one line for each skipped repository before the last
// line of a report, which is its total line.
func withSkipped(lines []string, skipped []project.Skipped) []string {
	if len(skipped) == 0 || len(lines) == 0 {
		return lines
	}
	out := append([]string{}, lines[:len(lines)-1]...)
	for _, s := range skipped {
		out = append(out, s.Line())
	}
	return append(out, lines[len(lines)-1])
}

// reportNames names the repositories a report covers: the repository
// arguments, or the list gh gives for --org, --user, or the logged-in user.
// The note is not empty when gh gave only the first part of that list.
func reportNames(o ui.Options, name string) ([]string, string, error) {
	if len(o.Repos) > 0 {
		return o.Repos, "", nil
	}
	who := o.Org
	if who == "" {
		who = o.User
	}
	names, err := gh.ListRepos(who)
	if err != nil {
		return nil, "", err
	}
	if len(names) >= gh.ListCap {
		return names, fmt.Sprintf("The %s covers the first %d repositories.", name, gh.ListCap), nil
	}
	return names, "", nil
}

// parseRepositories parses each name as a repository and drops a repeated one.
func parseRepositories(names []string) ([]project.Project, error) {
	var repositories []project.Project
	seen := map[string]bool{}
	for _, name := range names {
		p, err := project.Parse(name)
		if err != nil {
			return nil, err
		}
		if seen[p.NWO] {
			continue
		}
		seen[p.NWO] = true
		repositories = append(repositories, p)
	}
	return repositories, nil
}

// loadInventories reads the artifacts and the caches of each repository
// with gh. A repository with an error from gh, for the whole inventory or
// for a part of it, is skipped, because its numbers would be wrong.
func loadInventories(repositories []project.Project, wantArt, wantCache bool) ([]project.Project, []project.Skipped) {
	inventories := make([]project.Project, 0, len(repositories))
	var skipped []project.Skipped
	for _, base := range repositories {
		p, err := gh.Load(base, wantArt, wantCache)
		if err == nil && p.Err != "" {
			err = fmt.Errorf("%s", p.Err)
		}
		if err != nil {
			skipped = append(skipped, project.Skipped{NWO: base.NWO, Reason: err.Error()})
			continue
		}
		inventories = append(inventories, p)
	}
	return inventories, skipped
}

// chooseReport gives the report of the one report flag that is set, or nil.
// It refuses two report flags, and --artifact-list with --caches-only.
func chooseReport(summary, usage, artifactList, cacheOnly bool) (*report, error) {
	set := 0
	for _, on := range []bool{summary, usage, artifactList} {
		if on {
			set++
		}
	}
	if set > 1 {
		return nil, fmt.Errorf("use only one of --summary, --usage, or --artifact-list")
	}
	if artifactList && cacheOnly {
		return nil, fmt.Errorf("--artifact-list lists artifacts. Do not combine it with --caches-only")
	}
	switch {
	case summary:
		return &report{name: "summary", lines: project.SummaryLines}, nil
	case usage:
		return &report{name: "usage breakdown", lines: project.UsageLines}, nil
	case artifactList:
		return &report{name: "artifact list", lines: project.ArtifactLines}, nil
	}
	return nil, nil
}
