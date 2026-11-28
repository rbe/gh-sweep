// id: ADR-0019

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rbe/gh-sweep/internal/project"
)

// writePreferences writes the preferences file in a new config directory.
func writePreferences(t *testing.T, text string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	t.Cleanup(func() {
		if err := project.UsePreferences(project.DefaultPreferences()); err != nil {
			t.Fatal(err)
		}
	})
	path, err := project.PreferencesPath()
	if err != nil {
		t.Fatal(err)
	}
	if text == "" {
		return path
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTheReportsUseTheLargeSizeOfThePreferencesFile(t *testing.T) {
	mockInventory(t)
	writePreferences(t, `{"large_artifact_bytes": 1048576}`)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--artifact-list", "acme/widgets"}, &stdout, &stderr)
	if status != 0 || !strings.Contains(stdout.String(), "dist #101, workflow -, size 1.0 MB, created 2026-11-12, expires 2027-02-10, status active, large\n") ||
		strings.Contains(stdout.String(), "#100, workflow -, size 512 KB, created 2026-11-10, expires 2026-11-20, status expired, large") {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
}

func TestAWrongPreferencesFileStopsTheToolBeforeAnyGHCall(t *testing.T) {
	log := mockInventory(t)
	path := writePreferences(t, `{"old_artifact_days": 0}`)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--summary", "acme/widgets"}, &stdout, &stderr)
	if status != 1 || stdout.Len() != 0 || !strings.HasPrefix(stderr.String(), "gh-sweep: "+path+": old_artifact_days") {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(log); !os.IsNotExist(err) {
		t.Fatalf("gh was called: %v", err)
	}
}
