// id: ADR-0010

package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

// mockAccount puts the clock at 2026-11-28 and gives acme/widgets three
// releases, the user two codespaces, and the package app three versions.
func mockAccount(t *testing.T) string {
	t.Helper()
	log := mockInventory(t)
	mockAPI(t, "repos/acme/widgets/releases",
		fields("R", "3", "v3", "v3", "0", "0", "2026-11-20T00:00:00Z")+
			fields("A", "31", "app.tar.gz", "1048576")+
			fields("R", "2", "v2", "v2", "0", "1", "2026-06-01T00:00:00Z")+
			fields("A", "21", "app.tar.gz", "1048576")+
			fields("R", "1", "v1", "v1", "0", "0", "2026-01-01T00:00:00Z")+
			fields("A", "11", "app.tar.gz", "524288"))
	mockAPI(t, "user/codespaces",
		fields("old-space", "acme/widgets", "Shutdown", "2026-09-01T00:00:00Z", "0", "0")+
			fields("dirty-space", "acme/widgets", "Shutdown", "2026-09-01T00:00:00Z", "1", "0")+
			fields("live-space", "acme/widgets", "Available", "2026-11-27T00:00:00Z", "0", "0")+
			fields("lib-space", "other/lib", "Shutdown", "2026-09-01T00:00:00Z", "0", "0"))
	mockAPI(t, "user/packages/container/app/versions",
		fields("903", "sha256:c", "latest,1.2", "2026-11-20T00:00:00Z")+
			fields("902", "sha256:b", "", "2026-10-01T00:00:00Z")+
			fields("901", "sha256:a", "1.0", "2026-09-01T00:00:00Z"))
	day := time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)
	now = func() time.Time { return day }
	t.Cleanup(func() { now = time.Now })
	return log
}

func TestDeleteReleasesKeepsTheNewestAndTheTags(t *testing.T) {
	log := mockAccount(t)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--delete-releases", "--keep", "1", "acme/widgets"}, &stdout, &stderr)
	want := "acme/widgets: release 2 \"v2\", tag v2, created 2026-06-01, assets 1, size 1.0 MB, deleted\n" +
		"acme/widgets: release 1 \"v1\", tag v1, created 2026-01-01, assets 1, size 512 KB, deleted\n" +
		"total: deleted 2 releases, size 1.5 MB\n"
	if status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	calls := readLog(t, log)
	if !strings.Contains(calls, "DELETE repos/acme/widgets/releases/2\n") || strings.Contains(calls, "releases/3\n") ||
		strings.Contains(calls, "git/refs") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestDeleteReleaseAssetsKeepsTheReleaseAndSkipsAnImmutableRelease(t *testing.T) {
	log := mockAccount(t)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--delete-release-assets", "--older-than", "30", "acme/widgets"}, &stdout, &stderr)
	want := "acme/widgets: release v1 asset 11 \"app.tar.gz\", size 512 KB, deleted\n" +
		"total: deleted 1 release assets, size 512 KB\n"
	if status != 0 || stdout.String() != want {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	calls := readLog(t, log)
	if !strings.Contains(calls, "DELETE repos/acme/widgets/releases/assets/11\n") || strings.Contains(calls, "assets/21") ||
		strings.Contains(calls, "DELETE repos/acme/widgets/releases/1\n") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestDeletePackageVersionsDeletesOnlyOldUntaggedVersionsAndKeepsTheNewest(t *testing.T) {
	log := mockAccount(t)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--delete-package-versions", "--package", "app", "--untagged", "--mine"}, &stdout, &stderr)
	want := "user: container app version 902 sha256:b, tags -, created 2026-10-01, deleted\n" +
		"total: deleted 1 package versions\n"
	if status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	if calls := readLog(t, log); !strings.Contains(calls, "DELETE user/packages/container/app/versions/902\n") ||
		strings.Contains(calls, "versions/901") || strings.Contains(calls, "versions/903") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestDeletePackageVersionsOfAnOrganizationWithDryRunSendsNoDelete(t *testing.T) {
	log := mockAccount(t)
	mockAPI(t, "orgs/acme/packages/npm/@acme%2Fui/versions", fields("2", "1.0.1", "", "2026-11-01T00:00:00Z")+fields("1", "1.0.0", "", "2026-01-01T00:00:00Z"))

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--dry-run", "--delete-package-versions", "--package", "@acme/ui", "--package-type", "npm", "--older-than", "30", "--org", "acme"}, &stdout, &stderr)
	want := "acme: npm @acme/ui version 1 1.0.0, tags -, created 2026-01-01, would delete\n" +
		"total: would delete 1 package versions\n"
	if status != 0 || stdout.String() != want {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	if calls := readLog(t, log); strings.Contains(calls, "DELETE") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestDeleteCodespacesDeletesStoppedCodespacesWithNoUnsavedWork(t *testing.T) {
	log := mockAccount(t)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--delete-codespaces", "acme/widgets"}, &stdout, &stderr)
	want := "acme/widgets: codespace old-space, state Shutdown, last used 2026-09-01, deleted\n" +
		"kept: 1 stopped codespaces with uncommitted or unpushed changes\n" +
		"total: deleted 1 codespaces\n"
	if status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	calls := readLog(t, log)
	if !strings.Contains(calls, "DELETE user/codespaces/old-space\n") || strings.Contains(calls, "dirty-space") ||
		strings.Contains(calls, "live-space") || strings.Contains(calls, "lib-space") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestAccountFlagsRefuseAMissingTargetOrPackage(t *testing.T) {
	log := mockAccount(t)
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"--delete-releases", "acme/widgets"}, "gh-sweep: --delete-releases needs at least one of --older-than, --keep\n"},
		{[]string{"--delete-releases", "--keep", "2"}, "gh-sweep: to use --delete-releases, name the repositories, or pass --mine, --org, or --user\n"},
		{[]string{"--delete-package-versions", "--untagged", "--mine"}, "gh-sweep: --delete-package-versions needs --package NAME\n"},
		{[]string{"--delete-package-versions", "--package", "app", "--package-type", "pypi", "--untagged", "--mine"}, "gh-sweep: --package-type pypi is not one of container, npm, maven, rubygems, docker, nuget\n"},
		{[]string{"--delete-package-versions", "--package", "app", "--untagged"}, "gh-sweep: --delete-package-versions deletes packages of --mine or --org NAME. Pass one of them and no repository\n"},
		{[]string{"--delete-package-versions", "--package", "app", "--mine"}, "gh-sweep: --delete-package-versions needs at least one of --untagged, --older-than, --keep\n"},
		{[]string{"--delete-codespaces"}, "gh-sweep: --delete-codespaces deletes your own codespaces. Pass --mine, or name the repositories\n"},
		{[]string{"--delete-codespaces", "--org", "acme"}, "gh-sweep: --delete-codespaces deletes your own codespaces. Pass --mine, or name the repositories\n"},
	} {
		var stdout, stderr bytes.Buffer
		if status := Run(c.args, &stdout, &stderr); status != 2 || stdout.Len() != 0 || stderr.String() != c.want {
			t.Fatalf("%v: status %d\nstdout\n%sstderr\n%s", c.args, status, stdout.String(), stderr.String())
		}
	}
	if _, err := os.Stat(log); !os.IsNotExist(err) {
		t.Fatalf("gh was called: %v", err)
	}
}
