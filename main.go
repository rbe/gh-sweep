package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	opts, err := parseCLI(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "gh-sweep: %s\n", err)
		os.Exit(2)
	}
	prog := tea.NewProgram(newModel(opts), tea.WithAltScreen())
	final, err := prog.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gh-sweep: %s\n", err)
		os.Exit(1)
	}
	if m, ok := final.(model); ok {
		printSummary(m)
	}
}

func parseCLI(args []string) (options, error) {
	fs := flag.NewFlagSet("gh-sweep", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() { fmt.Fprint(os.Stderr, usageText) }
	org := fs.String("org", "", "list repositories in this organization")
	user := fs.String("user", "", "list repositories owned by this user")
	mine := fs.Bool("mine", false, "list repositories for the logged-in user")
	artOnly := fs.Bool("artifacts-only", false, "don't list or delete caches")
	cacheOnly := fs.Bool("caches-only", false, "don't list or delete artifacts")
	dry := fs.Bool("dry-run", false, "confirm without deleting")
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
		return options{}, err
	}
	repos = append(repos, fs.Args()...)
	if len(repos) > 0 && (*org != "" || *user != "" || *mine) {
		return options{}, fmt.Errorf("pass repositories or --org/--user/--mine, not both")
	}
	if *org != "" && (*user != "" || *mine) {
		return options{}, fmt.Errorf("use only one of --org, --user, or --mine")
	}
	if *user != "" && *mine {
		return options{}, fmt.Errorf("use only one of --user or --mine")
	}
	if *artOnly && *cacheOnly {
		return options{}, fmt.Errorf("use either --artifacts-only or --caches-only")
	}
	return options{
		repos:     repos,
		org:       *org,
		user:      *user,
		mine:      *mine,
		wantArt:   !*cacheOnly,
		wantCache: !*artOnly,
		dryRun:    *dry,
		filter:    *filter,
	}, nil
}

func printSummary(m model) {
	total := m.sessionArt + m.sessionCache
	if total == 0 {
		return
	}
	verb := "deleted"
	if m.opts.dryRun {
		verb = "would delete"
	}
	where := m.sessionRepos
	if where == "" {
		where = "GitHub"
	}
	fmt.Printf("gh-sweep: %s %d %s and %d %s from %s (%s)\n",
		verb,
		m.sessionArt, plural(m.sessionArt, "artifact", "artifacts"),
		m.sessionCache, plural(m.sessionCache, "cache", "caches"),
		where, humanSize(m.sessionBytes))
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
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
  -mine                list repositories for the logged-in user
  -artifacts-only      don't list or delete caches
  -caches-only         don't list or delete artifacts
  -filter text         start the item view with this filter
  -dry-run             confirm without deleting
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
