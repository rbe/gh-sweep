// id: ADR-0007

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSummarySkipsARepositoryThatFailsAndFinishesTheReport(t *testing.T) {
	mockInventory(t)
	t.Setenv("GH_MOCK_FAIL", "other/lib")

	var stdout, stderr bytes.Buffer
	status := run([]string{"--summary", "other/lib", "acme/widgets"}, &stdout, &stderr)
	want := "acme/widgets: artifacts 2, caches 1, size 3.5 MB\n" +
		"other/lib: skipped, HTTP 404: Not Found\n" +
		"total: artifacts 2, caches 1, size 3.5 MB\n"
	if status != 1 || stdout.String() != want || stderr.String() != "gh-sweep: the summary skipped 1 of 2 repositories\n" {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
}

func TestUsageSkipsARepositoryWhoseCachesFail(t *testing.T) {
	mockInventory(t)
	t.Setenv("GH_MOCK_CACHE_FAIL", "acme/widgets")

	var stdout, stderr bytes.Buffer
	status := run([]string{"--usage", "acme/widgets", "other/lib"}, &stdout, &stderr)
	want := "acme/widgets: skipped, HTTP 403: Resource not accessible by personal access token\n" +
		"total: size 0 B (0 bytes), artifacts 0 B, expired artifacts 0 B, caches 0 B\n"
	if status != 1 || stdout.String() != want || stderr.String() != "gh-sweep: the usage breakdown skipped 1 of 2 repositories\n" {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
}

func TestArtifactListSkipsARepositoryThatFailsAndListsTheNext(t *testing.T) {
	mockInventory(t)
	t.Setenv("GH_MOCK_FAIL", "other/lib")

	var stdout, stderr bytes.Buffer
	status := run([]string{"--artifact-list", "other/lib", "acme/widgets"}, &stdout, &stderr)
	want := "acme/widgets: dist #101, size 1.0 MB, created 2026-11-12, expires 2027-02-10, status active\n" +
		"acme/widgets: dist #100, size 512 KB, created 2026-11-10, expires 2026-11-20, status expired\n" +
		"other/lib: skipped, HTTP 404: Not Found\n" +
		"total: artifacts 2, size 1.5 MB, expired 1, expired size 512 KB\n"
	if status != 1 || stdout.String() != want || stderr.String() != "gh-sweep: the artifact list skipped 1 of 2 repositories\n" {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
}

func TestReportStopsBeforeTheFirstRepositoryWhenGHIsNotLoggedIn(t *testing.T) {
	root := t.TempDir()
	script := "#!/bin/sh\nprintf '%s\\n' 'gh: not logged in' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(root, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))

	var stdout, stderr bytes.Buffer
	status := run([]string{"--summary", "acme/widgets"}, &stdout, &stderr)
	if status != 1 || stdout.Len() != 0 || stderr.String() != "gh-sweep: gh is not logged in. Run: gh auth login\n" {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
}
