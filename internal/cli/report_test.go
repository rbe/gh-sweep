// id: ADR-0005

package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestUsagePrintsTheStorageOfEachRepositoryAndSendsNoDeleteRequest(t *testing.T) {
	log := mockInventory(t)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--usage", "other/lib", "acme/widgets"}, &stdout, &stderr)
	want := "acme/widgets: size 3.5 MB (3670016 bytes), artifacts 1.5 MB, expired artifacts 512 KB, caches 2.0 MB, share 100.0%\n" +
		"total: size 3.5 MB (3670016 bytes), artifacts 1.5 MB, expired artifacts 512 KB, caches 2.0 MB\n"
	if status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	if calls := readLog(t, log); strings.Contains(calls, "DELETE") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestArtifactListPrintsEachArtifactAndReadsNoCache(t *testing.T) {
	log := mockInventory(t)
	mockAPI(t, "repos/acme/widgets/actions/runs/500", "build\n")

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--artifact-list", "acme/widgets"}, &stdout, &stderr)
	want := "acme/widgets: dist #101, workflow build, size 1.0 MB, created 2026-11-12, expires 2027-02-10, status active\n" +
		"acme/widgets: dist #100, workflow -, size 512 KB, created 2026-11-10, expires 2026-11-20, status expired\n" +
		"total: artifacts 2, size 1.5 MB, expired 1, expired size 512 KB\n"
	if status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	calls := readLog(t, log)
	if strings.Contains(calls, "DELETE") || strings.Contains(calls, "LIST_CACHES") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestReportFlagsRefuseAnotherReportFlagAndArtifactListRefusesCachesOnly(t *testing.T) {
	log := mockInventory(t)
	for _, args := range [][]string{
		{"--summary", "--usage", "acme/widgets"},
		{"--usage", "--artifact-list", "acme/widgets"},
		{"--artifact-list", "--caches-only", "acme/widgets"},
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
