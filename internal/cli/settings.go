package cli

import (
	"fmt"
	"io"

	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
	"github.com/rbe/gh-sweep/internal/sweep"
	"github.com/rbe/gh-sweep/internal/ui"
)

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
	entries, listErr := sweep.SettingEntries(p, c)
	lines, _, err := sweep.Apply(entries, dryRun)
	if err != nil {
		return lines, err
	}
	return lines, listErr
}
