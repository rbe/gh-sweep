// id: ADR-0019

package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func usePreferences(t *testing.T, p Preferences) {
	t.Helper()
	if err := UsePreferences(p); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { current = DefaultPreferences() })
}

func TestLoadPreferencesGivesTheDefaultsWithNoFileAndKeepsTheDefaultOfAMissingKey(t *testing.T) {
	dir := t.TempDir()
	p, err := LoadPreferences(filepath.Join(dir, "none.json"))
	if err != nil || p != DefaultPreferences() {
		t.Fatalf("got %+v, %v", p, err)
	}
	path := filepath.Join(dir, "preferences.json")
	if err := os.WriteFile(path, []byte(`{"old_artifact_days": 10}`), 0o600); err != nil {
		t.Fatal(err)
	}
	p, err = LoadPreferences(path)
	want := DefaultPreferences()
	want.OldArtifactDays = 10
	if err != nil || p != want {
		t.Fatalf("got %+v, %v", p, err)
	}
}

func TestLoadPreferencesRefusesAnUnknownKeyAndAValueOutOfRange(t *testing.T) {
	dir := t.TempDir()
	for name, text := range map[string]string{
		"unknown.json": `{"old_days": 10}`,
		"range.json":   `{"keep_newest": 0}`,
		"broken.json":  `{"keep_newest": `,
	} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadPreferences(path); err == nil || !strings.HasPrefix(err.Error(), path+": ") {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestSavePreferencesWritesEveryKeyThatLoadReadsBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gh-sweep", "preferences.json")
	p := DefaultPreferences()
	p.LargeArtifactBytes = 10 << 20
	p.ConfirmEntries = 50
	if err := SavePreferences(path, p); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v, %v", info.Mode(), err)
	}
	data, _ := os.ReadFile(path)
	for _, f := range p.Fields() {
		if !strings.Contains(string(data), `"`+f.Key+`"`) {
			t.Fatalf("missing %s in\n%s", f.Key, data)
		}
	}
	if got, err := LoadPreferences(path); err != nil || got != p {
		t.Fatalf("got %+v, %v", got, err)
	}
	bad := DefaultPreferences()
	bad.RetentionDays = 0
	if err := SavePreferences(path, bad); err == nil {
		t.Fatal("a retention of 0 days was saved")
	}
}

func TestFieldsGiveTextThatWithFieldReadsBack(t *testing.T) {
	p := DefaultPreferences()
	p.LargeArtifactBytes = 1536
	p.ConfirmBytes = 3 << 30
	back := DefaultPreferences()
	for _, f := range p.Fields() {
		var err error
		if back, err = back.WithField(f.Key, f.Text); err != nil {
			t.Fatal(err)
		}
	}
	if back != p {
		t.Fatalf("got %+v, want %+v", back, p)
	}
	if _, err := p.WithField("keep_newest", "three"); err == nil {
		t.Fatal("text was read as a number")
	}
	if _, err := p.WithField("large_artifact_bytes", "5XB"); err == nil {
		t.Fatal("a wrong size was read")
	}
	if _, err := p.WithField("color", "red"); err == nil {
		t.Fatal("an unknown key was read")
	}
	if UsePreferences(Preferences{}) == nil {
		t.Fatal("zero preferences were put in force")
	}
}

func TestLargeAndOldFollowThePreferencesInForce(t *testing.T) {
	p := DefaultPreferences()
	p.LargeArtifactBytes = 1 << 20
	p.OldArtifactDays = 10
	usePreferences(t, p)
	now := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	it := Item{Kind: "artifact", Size: 1 << 20, When: now.Add(-11 * 24 * time.Hour)}
	if !it.Large() || !it.Old(now) {
		t.Fatalf("large %v, old %v", it.Large(), it.Old(now))
	}
	it.When = now.Add(-9 * 24 * time.Hour)
	it.Size--
	if it.Large() || it.Old(now) {
		t.Fatalf("large %v, old %v", it.Large(), it.Old(now))
	}
}
