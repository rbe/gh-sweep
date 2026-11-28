package main

import (
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
	"github.com/rbe/gh-sweep/internal/ui"
)

// now is the clock of the age filters. A test replaces it.
var now = time.Now

// job is one write flag that the tool runs in each named repository: its
// flag, the verb and the noun of its total line, the filter flags it reads,
// and the function that does it in one repository. The function gives its
// lines, the number of entries it changed or would change, and an error.
type job struct {
	flag        string
	verb        string
	noun        string
	filters     []string
	needsFilter bool
	each        func(p project.Project, f project.Filter, dryRun bool) ([]string, int, error)
}

// jobFlags holds the write flags and the filter flags of one command line.
type jobFlags struct {
	set    map[string]*bool
	order  []string
	jobs   map[string]job
	filter project.Filter
	fs     *flag.FlagSet
}

// filterFlags are the names of the filter flags.
var filterFlags = []string{"older-than", "branch", "status", "keep", "untagged"}

// registerJobs adds the write flags of the jobs and the filter flags.
func registerJobs(fs *flag.FlagSet, jobs []job) *jobFlags {
	jf := &jobFlags{set: map[string]*bool{}, jobs: map[string]job{}, fs: fs}
	for _, j := range jobs {
		name := strings.TrimPrefix(j.flag, "--")
		jf.set[name] = fs.Bool(name, false, j.verb+" "+j.noun+", then exit")
		jf.order = append(jf.order, name)
		jf.jobs[name] = j
	}
	fs.Func("older-than", "select only entries more than this number of days old", func(s string) error {
		days, err := strconv.Atoi(s)
		if err != nil || days < 1 {
			return fmt.Errorf("give a number of days of 1 or more")
		}
		jf.filter.OlderThan = days
		return nil
	})
	fs.StringVar(&jf.filter.Branch, "branch", "", "select only entries of this branch")
	fs.StringVar(&jf.filter.Status, "status", "", "select only workflow runs with this status or conclusion")
	fs.Func("keep", "keep this number of newest entries", func(s string) error {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			return fmt.Errorf("give a number of 0 or more")
		}
		jf.filter.Keep = n
		return nil
	})
	fs.BoolVar(&jf.filter.Untagged, "untagged", false, "select only untagged package versions")
	return jf
}

// choose gives the job of the one write flag that is set, or nil. It
// refuses two write flags, a filter flag that the job does not read, a
// filter flag with no write flag, and a job that needs a filter and has none.
func (jf *jobFlags) choose() (*job, error) {
	var chosen []string
	for _, name := range jf.order {
		if *jf.set[name] {
			chosen = append(chosen, "--"+name)
		}
	}
	if len(chosen) > 1 {
		return nil, fmt.Errorf("use only one of %s", strings.Join(chosen, ", "))
	}
	given := map[string]bool{}
	jf.fs.Visit(func(f *flag.Flag) { given[f.Name] = true })
	var j *job
	if len(chosen) == 1 {
		picked := jf.jobs[strings.TrimPrefix(chosen[0], "--")]
		j = &picked
	}
	filtered := false
	for _, name := range filterFlags {
		if !given[name] {
			continue
		}
		filtered = true
		if j == nil {
			return nil, fmt.Errorf("--%s selects entries for a write flag, such as --delete-runs. Pass one", name)
		}
		if !contains(j.filters, name) {
			return nil, fmt.Errorf("%s does not read --%s", j.flag, name)
		}
	}
	if j != nil && j.needsFilter && !filtered {
		return nil, fmt.Errorf("%s needs at least one of --%s", j.flag, strings.Join(j.filters, ", --"))
	}
	return j, nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// checkWriteTarget refuses a write flag that names no repository, no
// --org, no --user, and no --mine, because the default target is every
// repository of the logged-in user.
func checkWriteTarget(flagName string, named bool) error {
	if !named {
		return fmt.Errorf("to use %s, name the repositories, or pass --mine, --org, or --user", flagName)
	}
	return nil
}

// runJob runs the job in each named repository and prints its lines, then
// the total line. An error in one repository goes to stderr, and the next
// repository still runs.
func runJob(o ui.Options, j *job, f project.Filter, stdout, stderr io.Writer) error {
	if err := gh.Auth(); err != nil {
		return err
	}
	names, note, err := reportNames(o, "run of "+j.flag)
	if err != nil {
		return err
	}
	repositories, err := parseRepositories(names)
	if err != nil {
		return err
	}
	failed, total := 0, 0
	for _, p := range repositories {
		lines, n, err := j.each(p, f, o.DryRun)
		for _, line := range lines {
			fmt.Fprintln(stdout, line)
		}
		total += n
		if err != nil {
			fmt.Fprintf(stderr, "gh-sweep: %s: %s\n", p.NWO, err)
			failed++
		}
	}
	fmt.Fprintln(stdout, project.TotalLine(j.verb, j.noun, total, o.DryRun))
	if note != "" {
		fmt.Fprintf(stderr, "gh-sweep: %s\n", note)
	}
	if failed > 0 {
		return fmt.Errorf("%s failed in %d of %d repositories", j.flag, failed, len(repositories))
	}
	return nil
}
