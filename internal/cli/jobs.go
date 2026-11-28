package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
	"github.com/rbe/gh-sweep/internal/sweep"
	"github.com/rbe/gh-sweep/internal/ui"
)

// now is the clock of the age filters. A test replaces it.
var now = time.Now

// jobFlags holds the write flags and the filter flags of one command line.
type jobFlags struct {
	set     map[string]*bool
	order   []string
	actions map[string]sweep.Action
	filter  project.Filter
	fs      *flag.FlagSet
}

// filterFlags are the names of the filter flags.
var filterFlags = []string{"older-than", "branch", "status", "key-prefix", "keep", "untagged", "package", "package-type"}

// registerJobs adds the flag of each action and the filter flags.
func registerJobs(fs *flag.FlagSet, actions []sweep.Action) *jobFlags {
	jf := &jobFlags{set: map[string]*bool{}, actions: map[string]sweep.Action{}, fs: fs}
	for _, a := range actions {
		name := strings.TrimPrefix(a.Flag, "--")
		usage := a.Verb + " " + a.Noun + ", then exit"
		if a.ReadOnly {
			usage = "print the " + a.Noun + " of each repository, then exit"
		}
		jf.set[name] = fs.Bool(name, false, usage)
		jf.order = append(jf.order, name)
		jf.actions[name] = a
	}
	parse := func(name string) func(string) error {
		return func(v string) error { return sweep.SetFilter(&jf.filter, name, v) }
	}
	fs.Func("older-than", "select only entries more than this number of days old", parse("older-than"))
	fs.StringVar(&jf.filter.Branch, "branch", "", "select only entries of this branch")
	fs.StringVar(&jf.filter.Status, "status", "", "select only workflow runs with this status or conclusion")
	fs.Func("keep", "keep this number of newest entries", parse("keep"))
	fs.StringVar(&jf.filter.KeyPrefix, "key-prefix", "", "select only caches whose key starts with this text")
	fs.BoolVar(&jf.filter.Untagged, "untagged", false, "select only untagged package versions")
	fs.StringVar(&jf.filter.Package, "package", "", "the name of the package")
	fs.StringVar(&jf.filter.PackageType, "package-type", "container", "the type of the package: container, npm, maven, rubygems, docker, or nuget")
	return jf
}

// choose gives the action of the one write flag that is set, or nil, and
// the filter flags given. It refuses two write flags and a filter flag
// with no write flag.
func (jf *jobFlags) choose() (*sweep.Action, []string, error) {
	var chosen []string
	for _, name := range jf.order {
		if *jf.set[name] {
			chosen = append(chosen, "--"+name)
		}
	}
	if len(chosen) > 1 {
		return nil, nil, fmt.Errorf("use only one of %s", strings.Join(chosen, ", "))
	}
	visited := map[string]bool{}
	jf.fs.Visit(func(f *flag.Flag) { visited[f.Name] = true })
	var given []string
	for _, name := range filterFlags {
		if visited[name] {
			given = append(given, name)
		}
	}
	if len(chosen) == 0 {
		if len(given) > 0 {
			return nil, nil, fmt.Errorf("--%s selects entries for a write flag, such as --delete-runs. Pass one", given[0])
		}
		return nil, nil, nil
	}
	picked := jf.actions[strings.TrimPrefix(chosen[0], "--")]
	return &picked, given, nil
}

// runJob runs the action once for the account, or in each named
// repository, and prints its lines, then the total line. An error in one
// repository goes to stderr, and the next repository still runs.
func runJob(o ui.Options, a *sweep.Action, f project.Filter, stdout, stderr io.Writer) error {
	if err := gh.Auth(); err != nil {
		return err
	}
	if a.Account != nil {
		entries, err := a.Account(scopeOf(o), f, now())
		lines, t, applyErr := sweep.Apply(entries, o.DryRun)
		printLines(stdout, lines)
		fmt.Fprintln(stdout, a.Total(t, o.DryRun))
		if applyErr != nil {
			return applyErr
		}
		return err
	}
	names, note, err := reportNames(o, "run of "+a.Flag)
	if err != nil {
		return err
	}
	repositories, err := parseRepositories(names)
	if err != nil {
		return err
	}
	failed := 0
	var total project.Total
	for _, p := range repositories {
		entries, listErr := a.List(p, f, now())
		lines, t, err := sweep.Apply(entries, o.DryRun)
		printLines(stdout, lines)
		total = total.Add(t)
		if err == nil {
			err = listErr
		}
		if err != nil {
			fmt.Fprintf(stderr, "gh-sweep: %s: %s\n", p.NWO, err)
			failed++
		}
	}
	fmt.Fprintln(stdout, a.Total(total, o.DryRun))
	if note != "" {
		fmt.Fprintf(stderr, "gh-sweep: %s\n", note)
	}
	if failed > 0 {
		return fmt.Errorf("%s failed in %d of %d repositories", a.Flag, failed, len(repositories))
	}
	return nil
}

// scopeOf is the target of the command line.
func scopeOf(o ui.Options) sweep.Scope {
	return sweep.Scope{Repos: o.Repos, Org: o.Org, User: o.User, Mine: o.Mine}
}

func printLines(w io.Writer, lines []string) {
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
}
