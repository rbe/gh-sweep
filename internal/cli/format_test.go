// id: ADR-0017

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSummaryWritesJSONWithBytesAndAnEmptySkippedList(t *testing.T) {
	mockInventory(t)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--summary", "--format", "json", "acme/widgets"}, &stdout, &stderr)
	want := `{
  "report": "summary",
  "rows": [
    {
      "repository": "acme/widgets",
      "artifacts": 2,
      "caches": 1,
      "bytes": 3670016
    }
  ],
  "skipped": [],
  "total": {
    "artifacts": 2,
    "caches": 1,
    "bytes": 3670016
  }
}
`
	if status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
}

func TestSummaryWritesJSONWithNoRowsForARepositoryWithNoEntry(t *testing.T) {
	mockInventory(t)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--summary", "--format", "json", "other/lib"}, &stdout, &stderr)
	if status != 0 || !strings.Contains(stdout.String(), `"rows": [],`) || !strings.Contains(stdout.String(), `"skipped": [],`) {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
}

func TestUsageWritesCSVWithAHeaderAndNoTotalRecord(t *testing.T) {
	mockInventory(t)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--usage", "--format", "csv", "other/lib", "acme/widgets"}, &stdout, &stderr)
	want := "repository,bytes,artifact_bytes,expired_artifact_bytes,cache_bytes,share_percent,result,error\n" +
		"acme/widgets,3670016,1572864,524288,2097152,100,ok,\n"
	if status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
}

func TestArtifactListWritesJSONWithDatesAndTheSkippedRepository(t *testing.T) {
	mockInventory(t)
	t.Setenv("GH_MOCK_FAIL", "other/lib")

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--artifact-list", "--format", "json", "other/lib", "acme/widgets"}, &stdout, &stderr)
	for _, part := range []string{
		`"report": "artifact-list"`,
		`"id": "101",`,
		`"created": "2026-11-12T12:00:00Z",`,
		`"expires": "2027-02-10T12:00:00Z",`,
		`"status": "expired"`,
		`"repository": "other/lib",` + "\n      " + `"error": "HTTP 404: Not Found"`,
		`"expired_bytes": 524288`,
	} {
		if !strings.Contains(stdout.String(), part) {
			t.Fatalf("missing %s\nstdout\n%s", part, stdout.String())
		}
	}
	if status != 1 || stderr.String() != "gh-sweep: the artifact list skipped 1 of 2 repositories\n" {
		t.Fatalf("status %d\nstderr\n%s", status, stderr.String())
	}
}

func TestArtifactListWritesCSVWithASkippedRecord(t *testing.T) {
	mockInventory(t)
	t.Setenv("GH_MOCK_FAIL", "other/lib")

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--artifact-list", "--format", "csv", "other/lib", "acme/widgets"}, &stdout, &stderr)
	want := "repository,name,id,workflow,bytes,created,expires,status,large,result,error\n" +
		"acme/widgets,dist,101,,1048576,2026-11-12T12:00:00Z,2027-02-10T12:00:00Z,active,false,ok,\n" +
		"acme/widgets,dist,100,,524288,2026-11-10T12:00:00Z,2026-11-20T12:00:00Z,expired,false,ok,\n" +
		"other/lib,,,,,,,,,skipped,HTTP 404: Not Found\n"
	if status != 1 || stdout.String() != want {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
}

func TestOutputWritesTheReportToTheFileAndNothingToStandardOutput(t *testing.T) {
	mockInventory(t)
	path := filepath.Join(t.TempDir(), "summary.txt")

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--summary", "--output", path, "acme/widgets"}, &stdout, &stderr)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "acme/widgets: artifacts 2, caches 1, size 3.5 MB\ntotal: artifacts 2, caches 1, size 3.5 MB\n"
	if status != 0 || string(data) != want || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("status %d\nfile\n%sstdout\n%sstderr\n%s", status, data, stdout.String(), stderr.String())
	}
}

func TestFormatAndOutputRefuseAnUnknownFormatAndAMissingReportFlag(t *testing.T) {
	log := mockInventory(t)
	for _, args := range [][]string{
		{"--summary", "--format", "xml", "acme/widgets"},
		{"--format", "json", "acme/widgets"},
		{"--output", "out.json", "acme/widgets"},
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

func TestOutputFailsWithStatusOneWhenTheFileCannotBeWritten(t *testing.T) {
	mockInventory(t)
	path := filepath.Join(t.TempDir(), "missing", "summary.json")

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--summary", "--format", "json", "--output", path, "acme/widgets"}, &stdout, &stderr)
	if status != 1 || stdout.Len() != 0 || !strings.HasPrefix(stderr.String(), "gh-sweep: ") {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
}
