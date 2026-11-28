package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
	"github.com/rbe/gh-sweep/internal/ui"
)

// report is one report: its name in a note, the function that turns the
// loaded inventories into its lines, and the function that turns them into
// its table for JSON and CSV. With workflows set, the tool reads the
// workflow of each artifact before it writes the report.
type report struct {
	name      string
	workflows bool
	lines     func([]project.Project) []string
	table     func([]project.Project) project.Table
}

// reportOutput is where a report goes and in which format: text, json, or
// csv. An empty path is standard output.
type reportOutput struct {
	format string
	path   string
}

// reportFormats are the values of --format.
var reportFormats = []string{"text", "json", "csv"}

// checkReportOutput refuses an unknown format, and --format or --output
// without a report flag.
func checkReportOutput(out reportOutput, chosen *report, formatGiven bool) error {
	if (formatGiven || out.path != "") && chosen == nil {
		return fmt.Errorf("--format and --output need --summary, --usage, --artifact-list, or --advice")
	}
	for _, f := range reportFormats {
		if out.format == f {
			return nil
		}
	}
	return fmt.Errorf("--format takes %s, not %q", strings.Join(reportFormats, ", "), out.format)
}

// printReport loads the inventories and writes the lines of the report to
// stdout. A repository that cannot be loaded gets a skipped line before the
// total line, the other repositories still get their lines, and the result
// is an error that counts the skipped repositories.
func printReport(o ui.Options, r *report, out reportOutput, stdout, stderr io.Writer) error {
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
	inventories, skipped := loadInventories(repositories, o.WantArt, o.WantCache, r.workflows)
	if err := writeReport(r, out, inventories, skipped, stdout); err != nil {
		return err
	}
	if note != "" {
		fmt.Fprintf(stderr, "gh-sweep: %s\n", note)
	}
	if len(skipped) > 0 {
		return fmt.Errorf("the %s skipped %d of %d repositories", r.name, len(skipped), len(repositories))
	}
	return nil
}

// writeReport renders the report in the chosen format, then writes it to
// the file of --output, or to stdout when there is no file.
func writeReport(r *report, out reportOutput, inventories []project.Project, skipped []project.Skipped, stdout io.Writer) error {
	var body bytes.Buffer
	if err := renderReport(&body, r, out.format, inventories, skipped); err != nil {
		return err
	}
	if out.path == "" {
		_, err := stdout.Write(body.Bytes())
		return err
	}
	return os.WriteFile(out.path, body.Bytes(), 0o644)
}

// renderReport writes the report as text lines, as JSON, or as CSV.
func renderReport(w io.Writer, r *report, format string, inventories []project.Project, skipped []project.Skipped) error {
	switch format {
	case "json", "csv":
		t := r.table(inventories)
		t.Skipped = skipped
		if format == "json" {
			return t.WriteJSON(w)
		}
		return t.WriteCSV(w)
	}
	for _, line := range withSkipped(r.lines(inventories), skipped) {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
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
func loadInventories(repositories []project.Project, wantArt, wantCache, workflows bool) ([]project.Project, []project.Skipped) {
	inventories := make([]project.Project, 0, len(repositories))
	var skipped []project.Skipped
	for _, base := range repositories {
		p, err := gh.Load(base, wantArt, wantCache)
		if err == nil && p.Err != "" {
			err = fmt.Errorf("%s", p.Err)
		}
		if err == nil && workflows {
			err = gh.AddWorkflows(&p)
		}
		if err != nil {
			skipped = append(skipped, project.Skipped{NWO: base.NWO, Reason: err.Error()})
			continue
		}
		inventories = append(inventories, p)
	}
	return inventories, skipped
}

// reportFlags are the report flags of one command line and their filters.
// minSize and top are 0 when they were not given.
type reportFlags struct {
	summary, usage, artifactList, advice bool
	cacheOnly                            bool
	minSize                              int64
	minSizeGiven                         bool
	top                                  int
	topGiven                             bool
}

// chooseReport gives the report of the one report flag that is set, or nil.
// It refuses two report flags, --artifact-list with --caches-only,
// --min-size without --artifact-list, and --top without --usage.
func chooseReport(f reportFlags) (*report, error) {
	set := 0
	for _, on := range []bool{f.summary, f.usage, f.artifactList, f.advice} {
		if on {
			set++
		}
	}
	if set > 1 {
		return nil, fmt.Errorf("use only one of --summary, --usage, --artifact-list, or --advice")
	}
	if f.artifactList && f.cacheOnly {
		return nil, fmt.Errorf("--artifact-list lists artifacts. Do not combine it with --caches-only")
	}
	if f.minSizeGiven && !f.artifactList {
		return nil, fmt.Errorf("--min-size works only with --artifact-list")
	}
	if f.topGiven && !f.usage {
		return nil, fmt.Errorf("--top works only with --usage")
	}
	if f.topGiven && f.top < 1 {
		return nil, fmt.Errorf("--top takes a whole number of 1 or more, not %d", f.top)
	}
	switch {
	case f.summary:
		return &report{name: "summary", lines: project.SummaryLines, table: project.SummaryTable}, nil
	case f.usage:
		return usageReport(f.top), nil
	case f.artifactList:
		return artifactReport(f.minSize), nil
	case f.advice:
		return &report{name: "advice", lines: project.AdviceLines, table: project.AdviceTable}, nil
	}
	return nil, nil
}

// usageReport is the usage breakdown with the top largest repositories, or
// with every repository when top is 0.
func usageReport(top int) *report {
	return &report{
		name:  "usage breakdown",
		lines: func(inv []project.Project) []string { return project.TopUsageLines(inv, top) },
		table: func(inv []project.Project) project.Table { return project.TopUsageTable(inv, top) },
	}
}

// artifactReport is the artifact list with the artifacts of minBytes or
// more, and with the workflow of each artifact.
func artifactReport(minBytes int64) *report {
	return &report{
		name:      "artifact list",
		workflows: true,
		lines:     func(inv []project.Project) []string { return project.ArtifactLines(project.WithMinSize(inv, minBytes)) },
		table: func(inv []project.Project) project.Table {
			return project.ArtifactTable(project.WithMinSize(inv, minBytes))
		},
	}
}
