package sweep

import (
	"fmt"
	"strconv"

	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
)

// SettingActions are --retention-days and --actions-pr-approval. Each one
// reads one value, which the action screen asks for in its form.
var SettingActions = []Action{
	{Flag: "--retention-days", Title: "Set the artifact and log retention", Verb: "change", Noun: "retention settings",
		Filters: []string{"retention-days"}},
	{Flag: "--actions-pr-approval", Title: "Allow or deny pull requests from GitHub Actions", Verb: "change", Noun: "pull request settings",
		Filters: []string{"actions-pr-approval"}},
}

// SettingLabels name the value of each settings action.
var SettingLabels = map[string]string{
	"retention-days":      "retention, in days",
	"actions-pr-approval": "allow or deny",
}

// SetSetting parses the value of one settings flag into c. The command
// line and the action screen use it, so both refuse the same values.
func SetSetting(c *project.SettingsChange, name, value string) error {
	switch name {
	case "retention-days":
		days, err := strconv.Atoi(value)
		if err != nil || days < 1 {
			return fmt.Errorf("give a number of days of 1 or more")
		}
		c.RetentionDays = days
	case "actions-pr-approval":
		switch value {
		case "allow", "deny":
			allow := value == "allow"
			c.PRApproval = &allow
		default:
			return fmt.Errorf("give allow or deny")
		}
	default:
		return fmt.Errorf("--%s is not a setting", name)
	}
	return nil
}

// SettingEntries reads the settings of one repository and gives one entry
// for each setting of the change. A write sends the new value only when it
// differs. It stops at the first error and gives the entries before it.
func SettingEntries(p project.Project, c project.SettingsChange) ([]Entry, error) {
	var entries []Entry
	if c.RetentionDays > 0 {
		e, err := retentionEntry(p, c.RetentionDays)
		if err != nil {
			return entries, err
		}
		entries = append(entries, e)
	}
	if c.PRApproval != nil {
		e, err := prApprovalEntry(p, *c.PRApproval)
		if err != nil {
			return entries, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// retentionEntry reads the retention and checks the maximum.
func retentionEntry(p project.Project, days int) (Entry, error) {
	before, err := gh.GetRetention(p)
	if err != nil {
		return Entry{}, err
	}
	if err := project.CheckRetentionDays(days, before); err != nil {
		return Entry{}, err
	}
	e := counted(func(dry bool) string { return project.RetentionLine(p.NWO, before, days, dry) }, 0, nil)
	if before.Days == days {
		e.Total = project.Total{}
	} else {
		e.Apply = func() error { return gh.SetRetention(p, days) }
	}
	return e, nil
}

// prApprovalEntry reads the workflow permissions.
func prApprovalEntry(p project.Project, allow bool) (Entry, error) {
	before, err := gh.GetWorkflowPermissions(p)
	if err != nil {
		return Entry{}, err
	}
	e := counted(func(dry bool) string { return project.PRApprovalLine(p.NWO, before.CanApprovePRs, allow, dry) }, 0, nil)
	if before.CanApprovePRs == allow {
		e.Total = project.Total{}
	} else {
		after := before
		after.CanApprovePRs = allow
		e.Apply = func() error { return gh.SetWorkflowPermissions(p, after) }
	}
	return e, nil
}
