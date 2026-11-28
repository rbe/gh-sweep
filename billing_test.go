// id: ADR-0011

package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestBillingPrintsTheActionsItemsOfTheLoggedInUser(t *testing.T) {
	log := mockInventory(t)
	mockAPI(t, "user", "rbe\n")
	mockAPI(t, "users/rbe/settings/billing/usage/summary",
		fields("Actions", "actions_linux", "minutes", "1200", "9.6", "9.6", "0")+
			fields("Actions", "actions_storage", "gigabyte-hours", "350.5", "0.12", "0", "0.12"))

	var stdout, stderr bytes.Buffer
	status := run([]string{"--billing"}, &stdout, &stderr)
	want := "rbe: Actions actions_linux, 1200 minutes, gross $9.60, discount $9.60, net $0.00\n" +
		"rbe: Actions actions_storage, 350.5 gigabyte-hours, gross $0.12, discount $0.00, net $0.12\n" +
		"total: items 2, gross $9.72, net $0.12\n"
	if status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	if calls := readLog(t, log); !strings.Contains(calls, "GET users/rbe/settings/billing/usage/summary?product=Actions\n") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestBillingOfAnOrganizationReadsTheOrganizationReport(t *testing.T) {
	log := mockInventory(t)
	mockAPI(t, "organizations/acme/settings/billing/usage/summary", "")

	var stdout, stderr bytes.Buffer
	if status := run([]string{"--billing", "--org", "acme"}, &stdout, &stderr); status != 0 || stdout.String() != "total: items 0, gross $0.00, net $0.00\n" {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	calls := readLog(t, log)
	if !strings.Contains(calls, "GET organizations/acme/settings/billing/usage/summary?product=Actions\n") || strings.Contains(calls, "GET user\n") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestBillingPrintsTheErrorOfGitHubAndFails(t *testing.T) {
	mockInventory(t)
	mockAPI(t, "users/other/settings/billing/usage/summary.fail", "HTTP 403: Resource not accessible by personal access token\n")

	var stdout, stderr bytes.Buffer
	status := run([]string{"--billing", "--user", "other"}, &stdout, &stderr)
	if status != 1 || stderr.String() != "gh-sweep: other: HTTP 403: Resource not accessible by personal access token\n" {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
}

func TestBillingRefusesARepositoryArgument(t *testing.T) {
	log := mockInventory(t)

	var stdout, stderr bytes.Buffer
	status := run([]string{"--billing", "acme/widgets"}, &stdout, &stderr)
	if status != 2 || stderr.String() != "gh-sweep: --billing reads an account. Pass --user NAME, --org NAME, or no target for yourself\n" {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(log); !os.IsNotExist(err) {
		t.Fatalf("gh was called: %v", err)
	}
}
