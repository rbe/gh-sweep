package project

import "fmt"

// Retention is the artifact and log retention of a repository. GitHub keeps
// checks, workflow runs, commit statuses, artifacts, and logs for Days days.
// MaxDays is the largest value the repository accepts.
type Retention struct {
	Days    int
	MaxDays int
}

// WorkflowPermissions is the GITHUB_TOKEN setting of a repository. Default is
// "read" or "write". CanApprovePRs is the setting "Allow GitHub Actions to
// create and approve pull requests".
type WorkflowPermissions struct {
	Default       string
	CanApprovePRs bool
}

// SettingsChange is what one command line sets in each repository. A
// RetentionDays of 0 and a nil PRApproval leave that setting as it is.
type SettingsChange struct {
	RetentionDays int
	PRApproval    *bool
}

// Empty is true when the command line sets nothing.
func (c SettingsChange) Empty() bool {
	return c.RetentionDays == 0 && c.PRApproval == nil
}

// CheckRetentionDays refuses a value that the repository does not accept.
func CheckRetentionDays(days int, r Retention) error {
	if days > r.MaxDays {
		return fmt.Errorf("retention-days %d is above the maximum %d of this repository", days, r.MaxDays)
	}
	return nil
}

// RetentionLine is the line for one repository after --retention-days.
func RetentionLine(nwo string, before Retention, days int, dryRun bool) string {
	return fmt.Sprintf("%s: retention-days %d, %s (maximum %d)",
		nwo, before.Days, changeText(before.Days == days, dryRun, fmt.Sprint(days)), before.MaxDays)
}

// PRApprovalLine is the line for one repository after --actions-pr-approval.
func PRApprovalLine(nwo string, before, after, dryRun bool) string {
	return fmt.Sprintf("%s: actions-pr-approval %s, %s",
		nwo, approvalWord(before), changeText(before == after, dryRun, approvalWord(after)))
}

// changeText says whether the value stays, would change, or changed.
func changeText(same, dryRun bool, value string) string {
	switch {
	case same:
		return "unchanged"
	case dryRun:
		return "would set " + value
	}
	return "set " + value
}

// approvalWord is the flag value of the pull request setting.
func approvalWord(on bool) string {
	if on {
		return "allow"
	}
	return "deny"
}
