// Package cli is the command line of gh-sweep: it reads the flags,
// checks them, and runs a report, a settings change, an action, or the
// checklist.
package cli

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/rbe/gh-sweep/internal/project"
	"github.com/rbe/gh-sweep/internal/sweep"
	"github.com/rbe/gh-sweep/internal/ui"
)

// Run is the tool for one command line. It returns the exit status.
// With --summary, --usage, or --artifact-list it prints that report and does
// not open the checklist. With --retention-days or --actions-pr-approval it
// changes those settings and does not open the checklist.
func Run(args []string, stdout, stderr io.Writer) int {
	cmd, err := parseCLI(args)
	if err != nil {
		fmt.Fprintf(stderr, "gh-sweep: %s\n", err)
		return 2
	}
	if cmd.options.PreferencesPath, err = loadPreferences(); err != nil {
		fmt.Fprintf(stderr, "gh-sweep: %s\n", err)
		return 1
	}
	switch {
	case cmd.job != nil:
		err = runJob(cmd.options, cmd.job, cmd.filter, stdout, stderr)
	case !cmd.settings.Empty():
		err = applySettings(cmd.options, cmd.settings, stdout, stderr)
	case cmd.report != nil:
		err = printReport(cmd.options, cmd.report, cmd.output, stdout, stderr)
	default:
		err = ui.Run(cmd.options)
	}
	if err != nil {
		fmt.Fprintf(stderr, "gh-sweep: %s\n", err)
		return 1
	}
	return 0
}

// loadPreferences reads the preferences file and puts its values in force.
// It gives the path of the file, which the checklist writes.
func loadPreferences() (string, error) {
	path, err := project.PreferencesPath()
	if err != nil {
		return "", nil
	}
	p, err := project.LoadPreferences(path)
	if err != nil {
		return "", err
	}
	return path, project.UsePreferences(p)
}

// command is one parsed command line: the options of the checklist, the
// report to print, or nil for the checklist, and the settings to change.
type command struct {
	options  ui.Options
	report   *report
	output   reportOutput
	settings project.SettingsChange
	job      *sweep.Action
	filter   project.Filter
}

func parseCLI(args []string) (command, error) {
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
	advice := fs.Bool("advice", false, "print advice on what to delete, then exit")
	minSize := fs.String("min-size", "0", "with -artifact-list, show only artifacts of this size or more")
	top := fs.Int("top", 0, "with -usage, show only the largest n repositories")
	reportFormat := fs.String("format", "text", "write the report as text, json, or csv")
	output := fs.String("output", "", "write the report to this file instead of standard output")
	filter := fs.String("filter", "", "initial item filter")
	var settings project.SettingsChange
	fs.Func("retention-days", "set the artifact and log retention to this number of days, then exit", func(v string) error {
		return sweep.SetSetting(&settings, "retention-days", v)
	})
	fs.Func("actions-pr-approval", "allow or deny that GitHub Actions creates and approves pull requests, then exit", func(v string) error {
		return sweep.SetSetting(&settings, "actions-pr-approval", v)
	})
	jobs := registerJobs(fs, sweep.All())
	var repos []string
	fs.Func("repo", "repository owner/name (repeatable)", func(s string) error {
		repos = append(repos, s)
		return nil
	})
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		return command{}, err
	}
	repos = append(repos, fs.Args()...)
	if len(repos) > 0 && (*org != "" || *user != "" || *mine) {
		return command{}, fmt.Errorf("pass repositories or --org/--user/--mine, not both")
	}
	if *org != "" && (*user != "" || *mine) {
		return command{}, fmt.Errorf("use only one of --org, --user, or --mine")
	}
	if *user != "" && *mine {
		return command{}, fmt.Errorf("use only one of --user or --mine")
	}
	if *artOnly && *cacheOnly {
		return command{}, fmt.Errorf("use either --artifacts-only or --caches-only")
	}
	setFlags := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { setFlags[f.Name] = true })
	minBytes, err := project.ParseSize(*minSize)
	if err != nil {
		return command{}, err
	}
	chosen, err := chooseReport(reportFlags{
		summary: *summary, usage: *usage, artifactList: *artifactList, advice: *advice,
		cacheOnly: *cacheOnly,
		minSize:   minBytes, minSizeGiven: setFlags["min-size"],
		top: *top, topGiven: setFlags["top"],
	})
	if err != nil {
		return command{}, err
	}
	out := reportOutput{format: *reportFormat, path: *output}
	if err := checkReportOutput(out, chosen, setFlags["format"]); err != nil {
		return command{}, err
	}
	named := len(repos) > 0 || *org != "" || *user != "" || *mine
	if err := checkSettingsTarget(settings, chosen, named); err != nil {
		return command{}, err
	}
	j, given, err := jobs.choose()
	if err != nil {
		return command{}, err
	}
	s := sweep.Scope{Repos: repos, Org: *org, User: *user, Mine: *mine}
	if err := checkJob(j, given, jobs.filter, s, chosen, settings); err != nil {
		return command{}, err
	}
	wantCache := !*artOnly
	if *artifactList {
		wantCache = false
	}
	return command{
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
		output:   out,
		settings: settings,
		job:      j,
		filter:   jobs.filter,
	}, nil
}

// checkJob refuses a write flag together with a report flag or a settings
// flag, then checks the filters and the target of the action.
func checkJob(a *sweep.Action, given []string, f project.Filter, s sweep.Scope, chosen *report, c project.SettingsChange) error {
	if a == nil {
		return nil
	}
	if chosen != nil || !c.Empty() {
		return fmt.Errorf("%s does not combine with a report flag or a settings flag", a.Flag)
	}
	return a.CheckFilters(given, f, s)
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
