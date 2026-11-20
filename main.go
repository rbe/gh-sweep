package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
	"github.com/rbe/gh-sweep/internal/ui"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the tool for one command line. It returns the exit status.
// With --summary it prints the summary and does not open the checklist.
func run(args []string, stdout, stderr io.Writer) int {
	cmd, err := parseCLI(args)
	if err != nil {
		fmt.Fprintf(stderr, "gh-sweep: %s\n", err)
		return 2
	}
	if cmd.summary {
		err = printSummary(cmd.options, stdout, stderr)
	} else {
		err = ui.Run(cmd.options)
	}
	if err != nil {
		fmt.Fprintf(stderr, "gh-sweep: %s\n", err)
		return 1
	}
	return 0
}

// printSummary loads the inventories and writes the summary lines to stdout.
// It writes nothing to stdout when an inventory cannot be loaded.
func printSummary(o ui.Options, stdout, stderr io.Writer) error {
	if err := gh.Auth(); err != nil {
		return err
	}
	names, note, err := summaryNames(o)
	if err != nil {
		return err
	}
	repositories, err := parseRepositories(names)
	if err != nil {
		return err
	}
	inventories, err := loadInventories(repositories, o.WantArt, o.WantCache)
	if err != nil {
		return err
	}
	for _, line := range project.SummaryLines(inventories) {
		fmt.Fprintln(stdout, line)
	}
	if note != "" {
		fmt.Fprintf(stderr, "gh-sweep: %s\n", note)
	}
	return nil
}

// summaryNames names the repositories the summary covers: the repository
// arguments, or the list gh gives for --org, --user, or the logged-in user.
// The note is not empty when gh gave only the first part of that list.
func summaryNames(o ui.Options) ([]string, string, error) {
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
		return names, fmt.Sprintf("The summary covers the first %d repositories.", gh.ListCap), nil
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
// with gh. An error from gh, for the whole inventory or for a part of it,
// stops the load, because the numbers would be wrong.
func loadInventories(repositories []project.Project, wantArt, wantCache bool) ([]project.Project, error) {
	inventories := make([]project.Project, 0, len(repositories))
	for _, base := range repositories {
		p, err := gh.Load(base, wantArt, wantCache)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", base.NWO, err)
		}
		if p.Err != "" {
			return nil, fmt.Errorf("%s: %s", base.NWO, p.Err)
		}
		inventories = append(inventories, p)
	}
	return inventories, nil
}

// cli is one parsed command line: the options of the checklist, and
// whether --summary is set.
type cli struct {
	options ui.Options
	summary bool
}

func parseCLI(args []string) (cli, error) {
	fs := flag.NewFlagSet("gh-sweep", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() { fmt.Fprint(os.Stderr, usageText) }
	org := fs.String("org", "", "list repositories in this organization")
	user := fs.String("user", "", "list repositories owned by this user")
	mine := fs.Bool("mine", false, "list repositories for the logged-in user (the default)")
	artOnly := fs.Bool("artifacts-only", false, "don't list or delete caches")
	cacheOnly := fs.Bool("caches-only", false, "don't list or delete artifacts")
	dry := fs.Bool("dry-run", false, "confirm without deleting")
	summary := fs.Bool("summary", false, "print one line for each repository and a total line, then exit")
	filter := fs.String("filter", "", "initial item filter")
	var repos []string
	fs.Func("repo", "repository owner/name (repeatable)", func(s string) error {
		repos = append(repos, s)
		return nil
	})
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		return cli{}, err
	}
	repos = append(repos, fs.Args()...)
	if len(repos) > 0 && (*org != "" || *user != "" || *mine) {
		return cli{}, fmt.Errorf("pass repositories or --org/--user/--mine, not both")
	}
	if *org != "" && (*user != "" || *mine) {
		return cli{}, fmt.Errorf("use only one of --org, --user, or --mine")
	}
	if *user != "" && *mine {
		return cli{}, fmt.Errorf("use only one of --user or --mine")
	}
	if *artOnly && *cacheOnly {
		return cli{}, fmt.Errorf("use either --artifacts-only or --caches-only")
	}
	return cli{
		options: ui.Options{
			Repos:     repos,
			Org:       *org,
			User:      *user,
			Mine:      *mine,
			WantArt:   !*cacheOnly,
			WantCache: !*artOnly,
			DryRun:    *dry,
			Filter:    *filter,
		},
		summary: *summary,
	}, nil
}

const usageText = `gh-sweep — delete GitHub Actions artifacts and caches

The screen is a Charmbracelet (Bubble Tea) checklist. Mark repositories to
wipe them, or open one and check individual artifacts and cache entries.

Usage:
  gh-sweep [options] [owner/repo ...]

Options:
  -repo owner/repo     repository, repeatable
  -org name            list repositories in this organization
  -user name           list repositories owned by this user
  -mine                list repositories for the logged-in user (the default)
  -artifacts-only      don't list or delete caches
  -caches-only         don't list or delete artifacts
  -filter text         start the item view with this filter
  -dry-run             confirm without deleting
  -summary             print one line for each repository and a total line, then exit
  -h                   show this help

Project keys:
  space       mark a repository for a full wipe
  enter       open it and pick artifacts and caches
  a / n       mark all shown, or clear them
  d           delete everything marked
  s           scan sizes
  +           add owner/repo
  /           filter
  q           quit

Inside a repository:
  space       toggle the row (a group toggles every row in it)
  enter       toggle an item, or open and close a group
  j / k       move
  h / l       collapse or expand    e / c fold all
  x           select expired artifacts
  o           select artifacts older than N days
  t           keep the newest N of each name, select the rest
  d           delete the checked rows
  b           back to the project list
  q           quit
`
