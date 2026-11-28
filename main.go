package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
	"github.com/rbe/gh-sweep/internal/ui"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the tool for one command line. It returns the exit status.
// With --summary, --usage, or --artifact-list it prints that report and does
// not open the checklist. With --retention-days or --actions-pr-approval it
// changes those settings and does not open the checklist.
func run(args []string, stdout, stderr io.Writer) int {
	cmd, err := parseCLI(args)
	if err != nil {
		fmt.Fprintf(stderr, "gh-sweep: %s\n", err)
		return 2
	}
	switch {
	case cmd.job != nil:
		err = runJob(cmd.options, cmd.job, cmd.filter, stdout, stderr)
	case !cmd.settings.Empty():
		err = applySettings(cmd.options, cmd.settings, stdout, stderr)
	case cmd.report != nil:
		err = printReport(cmd.options, cmd.report, stdout, stderr)
	default:
		err = ui.Run(cmd.options)
	}
	if err != nil {
		fmt.Fprintf(stderr, "gh-sweep: %s\n", err)
		return 1
	}
	return 0
}

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

// cli is one parsed command line: the options of the checklist, the
// report to print, or nil for the checklist, and the settings to change.
type cli struct {
	options  ui.Options
	report   *report
	settings project.SettingsChange
	job      *job
	filter   project.Filter
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
	usage := fs.Bool("usage", false, "print the storage of each repository, largest first, then exit")
	artifactList := fs.Bool("artifact-list", false, "print one line for each artifact with size, dates, and status, then exit")
	filter := fs.String("filter", "", "initial item filter")
	var settings project.SettingsChange
	fs.Func("retention-days", "set the artifact and log retention to this number of days, then exit", func(s string) error {
		days, err := strconv.Atoi(s)
		if err != nil || days < 1 {
			return fmt.Errorf("give a number of days of 1 or more")
		}
		settings.RetentionDays = days
		return nil
	})
	fs.Func("actions-pr-approval", "allow or deny that GitHub Actions creates and approves pull requests, then exit", func(s string) error {
		switch s {
		case "allow", "deny":
			allow := s == "allow"
			settings.PRApproval = &allow
			return nil
		}
		return fmt.Errorf("give allow or deny")
	})
	jobs := registerJobs(fs, allJobs())
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
	chosen, err := chooseReport(*summary, *usage, *artifactList, *cacheOnly)
	if err != nil {
		return cli{}, err
	}
	named := len(repos) > 0 || *org != "" || *user != "" || *mine
	if err := checkSettingsTarget(settings, chosen, named); err != nil {
		return cli{}, err
	}
	j, err := jobs.choose()
	if err != nil {
		return cli{}, err
	}
	o := ui.Options{Repos: repos, Org: *org, User: *user, Mine: *mine}
	if err := checkJob(j, jobs.filter, o, chosen, settings, named); err != nil {
		return cli{}, err
	}
	wantCache := !*artOnly
	if *artifactList {
		wantCache = false
	}
	return cli{
		options: ui.Options{
			Repos:     repos,
			Org:       *org,
			User:      *user,
			Mine:      *mine,
			WantArt:   !*cacheOnly,
			WantCache: wantCache,
			DryRun:    *dry,
			Filter:    *filter,
		},
		report:   chosen,
		settings: settings,
		job:      j,
		filter:   jobs.filter,
	}, nil
}

// allJobs are the write flags that run in each named repository.
func allJobs() []job {
	jobs := append(append([]job{}, runJobs...), cacheJobs...)
	jobs = append(jobs, accountJobs...)
	jobs = append(jobs, billingJob, branchJob)
	jobs = append(jobs, deploymentJobs...)
	return append(jobs, staleJobs...)
}

// checkJob refuses a write flag together with a report flag or a settings
// flag, a write flag with no named target, and a status that GitHub does
// not accept.
func checkJob(j *job, f project.Filter, o ui.Options, chosen *report, c project.SettingsChange, named bool) error {
	if j == nil {
		return nil
	}
	if chosen != nil || !c.Empty() {
		return fmt.Errorf("%s does not combine with a report flag or a settings flag", j.flag)
	}
	switch {
	case j.target != nil:
		if err := j.target(o, f); err != nil {
			return err
		}
	case !j.readOnly:
		if err := checkWriteTarget(j.flag, named); err != nil {
			return err
		}
	}
	return project.CheckRunStatus(f.Status)
}

// checkSettingsTarget refuses a settings change together with a report, and
// a settings change that names no repository, no --org, no --user, and no
// --mine.
func checkSettingsTarget(c project.SettingsChange, chosen *report, named bool) error {
	if c.Empty() {
		return nil
	}
	if chosen != nil {
		return fmt.Errorf("--retention-days and --actions-pr-approval do not combine with --summary, --usage, or --artifact-list")
	}
	if !named {
		return fmt.Errorf("to change settings, name the repositories, or pass --mine, --org, or --user")
	}
	return nil
}

// applySettings changes the settings of each named repository and prints one
// line for each setting. An error in one repository goes to stderr, and the
// next repository still gets its change.
func applySettings(o ui.Options, c project.SettingsChange, stdout, stderr io.Writer) error {
	if err := gh.Auth(); err != nil {
		return err
	}
	names, note, err := reportNames(o, "settings change")
	if err != nil {
		return err
	}
	repositories, err := parseRepositories(names)
	if err != nil {
		return err
	}
	failed := 0
	for _, p := range repositories {
		lines, err := changeSettings(p, c, o.DryRun)
		for _, line := range lines {
			fmt.Fprintln(stdout, line)
		}
		if err != nil {
			fmt.Fprintf(stderr, "gh-sweep: %s: %s\n", p.NWO, err)
			failed++
		}
	}
	if note != "" {
		fmt.Fprintf(stderr, "gh-sweep: %s\n", note)
	}
	if failed > 0 {
		return fmt.Errorf("the settings of %d of %d repositories did not change", failed, len(repositories))
	}
	return nil
}

// changeSettings changes the settings of one repository. It stops at the
// first error and gives the lines of the settings done before it.
func changeSettings(p project.Project, c project.SettingsChange, dryRun bool) ([]string, error) {
	var lines []string
	if c.RetentionDays > 0 {
		line, err := changeRetention(p, c.RetentionDays, dryRun)
		if err != nil {
			return lines, err
		}
		lines = append(lines, line)
	}
	if c.PRApproval != nil {
		line, err := changePRApproval(p, *c.PRApproval, dryRun)
		if err != nil {
			return lines, err
		}
		lines = append(lines, line)
	}
	return lines, nil
}

// changeRetention reads the retention, checks the maximum, and writes the
// new value when it differs and dryRun is false.
func changeRetention(p project.Project, days int, dryRun bool) (string, error) {
	before, err := gh.GetRetention(p)
	if err != nil {
		return "", err
	}
	if err := project.CheckRetentionDays(days, before); err != nil {
		return "", err
	}
	if !dryRun && before.Days != days {
		if err := gh.SetRetention(p, days); err != nil {
			return "", err
		}
	}
	return project.RetentionLine(p.NWO, before, days, dryRun), nil
}

// changePRApproval reads the workflow permissions and writes the new pull
// request setting when it differs and dryRun is false.
func changePRApproval(p project.Project, allow, dryRun bool) (string, error) {
	before, err := gh.GetWorkflowPermissions(p)
	if err != nil {
		return "", err
	}
	if !dryRun && before.CanApprovePRs != allow {
		after := before
		after.CanApprovePRs = allow
		if err := gh.SetWorkflowPermissions(p, after); err != nil {
			return "", err
		}
	}
	return project.PRApprovalLine(p.NWO, before.CanApprovePRs, allow, dryRun), nil
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
  -usage               print the storage of each repository, largest first, then exit
  -artifact-list       print one line for each artifact with size, dates, and status, then exit
  -retention-days n    set the artifact and log retention of each repository to n days, then exit
  -actions-pr-approval allow|deny
                       allow or deny that GitHub Actions creates and approves pull requests, then exit
  -delete-runs         delete completed workflow runs and their logs, then exit
  -delete-run-logs     delete the logs of completed workflow runs and keep the runs, then exit
  -delete-caches       delete the caches that the filters select, then exit
  -cache-usage         print the cache usage and the cache limits of each repository, then exit
  -delete-releases     delete the releases that the filters select and keep their tags, then exit
  -delete-release-assets
                       delete the assets of the releases that the filters select, then exit
  -delete-package-versions
                       delete versions of the package that -package names, then exit
  -delete-codespaces   delete your stopped codespaces with no unsaved work, then exit
  -billing             print the billed GitHub Actions usage of this month, then exit
  -delete-merged-branches
                       delete branches whose pull request is merged or closed, then exit
  -delete-deployments  delete old deployments and keep the newest of each environment, then exit
  -delete-unused-environments
                       delete environments with no deployment, then exit
  -stale-list          print each repository with its last push and disk usage, oldest first, then exit
  -archive             archive the named repositories, then exit

Filters of the write flags:
  -older-than n        only entries more than n days old
  -branch name         only workflow runs or caches of this branch
  -status value        only workflow runs with this status or conclusion, such as failure
  -key-prefix text     only caches whose key starts with this text
  -keep n              keep the newest n releases, package versions, or deployments of each environment
  -untagged            only package versions with no tag
  -package name        the package of -delete-package-versions
  -package-type type   container (the default), npm, maven, rubygems, docker, or nuget
  -h                   show this help

Project keys:
  space       mark a repository for a full wipe
  enter       open it and pick artifacts and caches
  a / n       mark all shown, or clear them
  d           delete everything marked
  s           scan sizes
  u           storage usage of the scanned projects
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
  i           details of each artifact
  d           delete the checked rows
  b           back to the project list
  q           quit
`
