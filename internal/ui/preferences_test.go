// id: ADR-0019

package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rbe/gh-sweep/internal/project"
)

func restorePreferences(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		if err := project.UsePreferences(project.DefaultPreferences()); err != nil {
			t.Fatal(err)
		}
	})
}

func TestPreferencesScreenSavesTheValuesAndPutsThemInForce(t *testing.T) {
	restorePreferences(t)
	path := filepath.Join(t.TempDir(), "preferences.json")
	m := actionModel(Options{PreferencesPath: path}, "acme/widgets")
	m = press(t, m, "p")
	if m.screen != scrPreferences || !strings.Contains(stripANSI(m.View()), "old artifact, days") {
		t.Fatalf("view\n%s", stripANSI(m.View()))
	}
	m = press(t, m, "backspace", "9", "enter")
	if m.screen != scrProjects || project.Prefs().OldArtifactDays != 9 {
		t.Fatalf("screen %v, prefs %+v, status %q", m.screen, project.Prefs(), m.status)
	}
	saved, err := project.LoadPreferences(path)
	if err != nil || saved.OldArtifactDays != 9 {
		t.Fatalf("saved %+v, %v", saved, err)
	}
	m.screen = scrItems
	m.open = 0
	m = press(t, m, "o")
	if m.input.Value() != "9" {
		t.Fatalf("the o prompt starts with %q", m.input.Value())
	}
}

func TestPreferencesScreenKeepsAWrongValueOutAndEscChangesNothing(t *testing.T) {
	restorePreferences(t)
	m := actionModel(Options{}, "acme/widgets")
	m = press(t, m, "p", "tab", "x", "enter")
	if m.screen != scrPreferences || !strings.Contains(m.status, "large_artifact_bytes") || project.Prefs() != project.DefaultPreferences() {
		t.Fatalf("screen %v, status %q", m.screen, m.status)
	}
	m = press(t, m, "esc")
	if m.screen != scrProjects || project.Prefs() != project.DefaultPreferences() {
		t.Fatalf("screen %v, prefs %+v", m.screen, project.Prefs())
	}
	m = press(t, m, "p", "shift+tab", "backspace", "backspace", "4", "enter")
	if project.Prefs().ConfirmBytes != 14 || m.status != "The preferences apply until gh-sweep quits" {
		t.Fatalf("prefs %+v, status %q", project.Prefs(), m.status)
	}
}

func TestPreferencesScreenRefusesAValueOutOfRangeAndAFileItCannotWrite(t *testing.T) {
	restorePreferences(t)
	m := actionModel(Options{}, "acme/widgets")
	m = press(t, m, "p", "backspace", "0", "enter")
	if m.screen != scrPreferences || !strings.HasPrefix(m.status, "old_artifact_days must be from 1 to 400") {
		t.Fatalf("screen %v, status %q", m.screen, m.status)
	}
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	m = actionModel(Options{PreferencesPath: filepath.Join(file, "preferences.json")}, "acme/widgets")
	m = press(t, m, "p", "enter")
	if m.screen != scrPreferences || !strings.HasPrefix(m.status, "Could not save the preferences: ") {
		t.Fatalf("screen %v, status %q", m.screen, m.status)
	}
	if !strings.Contains(stripANSI(m.View()), "Could not save the preferences") {
		t.Fatalf("view\n%s", stripANSI(m.View()))
	}
}
