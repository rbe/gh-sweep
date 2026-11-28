// id: ADR-0006

package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRetentionDaysSetsTheRetentionOfEachRepositoryAndPrintsTheOldValue(t *testing.T) {
	log := mockInventory(t)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--retention-days", "30", "acme/widgets", "other/lib"}, &stdout, &stderr)
	want := "acme/widgets: retention-days 90, set 30 (maximum 90)\n" +
		"other/lib: retention-days 90, set 30 (maximum 90)\n"
	if status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	calls := readLog(t, log)
	for _, call := range []string{"PUT_RETENTION acme/widgets days=30", "PUT_RETENTION other/lib days=30"} {
		if !strings.Contains(calls, call) {
			t.Fatalf("no %q in the gh calls\n%s", call, calls)
		}
	}
	if strings.Contains(calls, "LIST_") || strings.Contains(calls, "DELETE") || strings.Contains(calls, "PUT_WORKFLOW") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestActionsPRApprovalKeepsTheDefaultPermissionAndSendsNoRequestWhenUnchanged(t *testing.T) {
	log := mockInventory(t)
	t.Setenv("GH_MOCK_WORKFLOW", "write false")

	var stdout, stderr bytes.Buffer
	if status := Run([]string{"--actions-pr-approval", "allow", "acme/widgets"}, &stdout, &stderr); status != 0 {
		t.Fatalf("status %d\n%s", status, stderr.String())
	}
	if stdout.String() != "acme/widgets: actions-pr-approval deny, set allow\n" {
		t.Fatalf("stdout\n%s", stdout.String())
	}
	if calls := readLog(t, log); !strings.Contains(calls, "PUT_WORKFLOW acme/widgets default_workflow_permissions=write can_approve_pull_request_reviews=true") {
		t.Fatalf("gh calls\n%s", calls)
	}

	stdout.Reset()
	if status := Run([]string{"--actions-pr-approval", "deny", "other/lib"}, &stdout, &stderr); status != 0 {
		t.Fatalf("status %d\n%s", status, stderr.String())
	}
	if stdout.String() != "other/lib: actions-pr-approval deny, unchanged\n" {
		t.Fatalf("stdout\n%s", stdout.String())
	}
	if calls := readLog(t, log); strings.Contains(calls, "PUT_WORKFLOW other/lib") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestSettingsWithDryRunReadAndSendNoChange(t *testing.T) {
	log := mockInventory(t)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--dry-run", "--retention-days", "7", "--actions-pr-approval", "allow", "acme/widgets"}, &stdout, &stderr)
	want := "acme/widgets: retention-days 90, would set 7 (maximum 90)\n" +
		"acme/widgets: actions-pr-approval deny, would set allow\n"
	if status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	if calls := readLog(t, log); strings.Contains(calls, "PUT") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestRetentionDaysAboveTheMaximumChangesNothingInThatRepository(t *testing.T) {
	log := mockInventory(t)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--retention-days", "400", "acme/widgets"}, &stdout, &stderr)
	want := "gh-sweep: acme/widgets: retention-days 400 is above the maximum 90 of this repository\n" +
		"gh-sweep: the settings of 1 of 1 repositories did not change\n"
	if status != 1 || stdout.Len() != 0 || stderr.String() != want {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	if calls := readLog(t, log); strings.Contains(calls, "PUT") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestSettingsErrorInOneRepositoryStillChangesTheNextAndFails(t *testing.T) {
	log := mockInventory(t)
	t.Setenv("GH_MOCK_SETTINGS_FAIL", "acme/widgets")

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--retention-days", "30", "acme/widgets", "other/lib"}, &stdout, &stderr)
	wantErr := "gh-sweep: acme/widgets: HTTP 404: Not Found\n" +
		"gh-sweep: the settings of 1 of 2 repositories did not change\n"
	if status != 1 || stdout.String() != "other/lib: retention-days 90, set 30 (maximum 90)\n" || stderr.String() != wantErr {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	if calls := readLog(t, log); !strings.Contains(calls, "PUT_RETENTION other/lib days=30") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestSettingsWithNoArgumentCoverTheLoggedInUserOnlyWithMine(t *testing.T) {
	log := mockInventory(t)
	t.Setenv("GH_MOCK_REPOS", "acme/widgets\nother/lib")

	var stdout, stderr bytes.Buffer
	if status := Run([]string{"--retention-days", "30"}, &stdout, &stderr); status != 2 || stdout.Len() != 0 ||
		stderr.String() != "gh-sweep: to change settings, name the repositories, or pass --mine, --org, or --user\n" {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(log); !os.IsNotExist(err) {
		t.Fatalf("gh was called: %v", err)
	}

	stderr.Reset()
	if status := Run([]string{"--mine", "--retention-days", "30"}, &stdout, &stderr); status != 0 {
		t.Fatalf("status %d\n%s", status, stderr.String())
	}
	calls := readLog(t, log)
	for _, call := range []string{"CALL\trepo\tlist\t--limit", "PUT_RETENTION acme/widgets days=30", "PUT_RETENTION other/lib days=30"} {
		if !strings.Contains(calls, call) {
			t.Fatalf("no %q in the gh calls\n%s", call, calls)
		}
	}
}

func TestSettingsFlagsRefuseABadValueAndAReportFlag(t *testing.T) {
	log := mockInventory(t)
	for _, args := range [][]string{
		{"--retention-days", "0", "acme/widgets"},
		{"--retention-days", "ten", "acme/widgets"},
		{"--actions-pr-approval", "yes", "acme/widgets"},
		{"--summary", "--retention-days", "30", "acme/widgets"},
		{"--usage", "--actions-pr-approval", "deny", "acme/widgets"},
	} {
		var stdout, stderr bytes.Buffer
		if status := Run(args, &stdout, &stderr); status != 2 || stdout.Len() != 0 || !strings.HasPrefix(stderr.String(), "gh-sweep: ") {
			t.Fatalf("%v: status %d\nstdout\n%sstderr\n%s", args, status, stdout.String(), stderr.String())
		}
	}
	if _, err := os.Stat(log); !os.IsNotExist(err) {
		t.Fatalf("gh was called: %v", err)
	}
}
