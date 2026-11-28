package project

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
)

// Preferences are the values a person can change in the checklist: the
// limits of an old and a large artifact, the count of the advice, the
// defaults of the prompts and of the retention form, and the limits of the
// typed confirmation.
type Preferences struct {
	OldArtifactDays       int   `json:"old_artifact_days"`
	LargeArtifactBytes    int64 `json:"large_artifact_bytes"`
	AdviceTopRepositories int   `json:"advice_top_repositories"`
	KeepNewest            int   `json:"keep_newest"`
	RetentionDays         int   `json:"retention_days"`
	ConfirmEntries        int   `json:"confirm_entries"`
	ConfirmBytes          int64 `json:"confirm_bytes"`
}

// DefaultPreferences are the values before a person changes one.
func DefaultPreferences() Preferences {
	return Preferences{
		OldArtifactDays:       5,
		LargeArtifactBytes:    50 << 20,
		AdviceTopRepositories: 3,
		KeepNewest:            3,
		RetentionDays:         90,
		ConfirmEntries:        20,
		ConfirmBytes:          1 << 30,
	}
}

// current holds the preferences in force. UsePreferences sets it, and
// Prefs reads it.
var current = DefaultPreferences()

// Prefs gives the preferences in force.
func Prefs() Preferences { return current }

// UsePreferences puts the preferences in force after it checks them.
func UsePreferences(p Preferences) error {
	if err := p.Check(); err != nil {
		return err
	}
	current = p
	return nil
}

// PreferenceField is one value of Preferences as the checklist shows it:
// its key in the file, its label, and its text.
type PreferenceField struct {
	Key   string
	Label string
	Text  string
}

// Fields gives the values in a fixed order as text. A size is text such
// as 50.0 MB, which ParseSize reads back.
func (p Preferences) Fields() []PreferenceField {
	return []PreferenceField{
		{"old_artifact_days", "old artifact, days", strconv.Itoa(p.OldArtifactDays)},
		{"large_artifact_bytes", "large artifact, size", sizeText(p.LargeArtifactBytes)},
		{"advice_top_repositories", "advice, repositories", strconv.Itoa(p.AdviceTopRepositories)},
		{"keep_newest", "keep newest (t)", strconv.Itoa(p.KeepNewest)},
		{"retention_days", "retention, days", strconv.Itoa(p.RetentionDays)},
		{"confirm_entries", "type to confirm, entries", strconv.Itoa(p.ConfirmEntries)},
		{"confirm_bytes", "type to confirm, size", sizeText(p.ConfirmBytes)},
	}
}

// sizeText is a size without a fraction in the largest unit that fits, so
// that ParseSize reads the same number back.
func sizeText(b int64) string {
	for _, u := range sizeUnits {
		if u.bytes > 1 && b > 0 && b%u.bytes == 0 {
			return fmt.Sprintf("%d%s", b/u.bytes, u.suffix)
		}
	}
	return strconv.FormatInt(b, 10)
}

// WithField gives a copy with the value of the key read from text.
func (p Preferences) WithField(key, text string) (Preferences, error) {
	number := func() (int, error) {
		n, err := strconv.Atoi(text)
		if err != nil {
			return 0, fmt.Errorf("%s: %q is not a whole number", key, text)
		}
		return n, nil
	}
	size := func() (int64, error) {
		n, err := ParseSize(text)
		if err != nil {
			return 0, fmt.Errorf("%s: %v", key, err)
		}
		return n, nil
	}
	var err error
	switch key {
	case "old_artifact_days":
		p.OldArtifactDays, err = number()
	case "large_artifact_bytes":
		p.LargeArtifactBytes, err = size()
	case "advice_top_repositories":
		p.AdviceTopRepositories, err = number()
	case "keep_newest":
		p.KeepNewest, err = number()
	case "retention_days":
		p.RetentionDays, err = number()
	case "confirm_entries":
		p.ConfirmEntries, err = number()
	case "confirm_bytes":
		p.ConfirmBytes, err = size()
	default:
		err = fmt.Errorf("no preference %q", key)
	}
	return p, err
}

// Check refuses a value outside its range.
func (p Preferences) Check() error {
	checks := []struct {
		key      string
		value    int64
		min, max int64
	}{
		{"old_artifact_days", int64(p.OldArtifactDays), 1, 400},
		{"large_artifact_bytes", p.LargeArtifactBytes, 1, 1 << 40},
		{"advice_top_repositories", int64(p.AdviceTopRepositories), 1, 50},
		{"keep_newest", int64(p.KeepNewest), 1, 1000},
		{"retention_days", int64(p.RetentionDays), 1, 400},
		{"confirm_entries", int64(p.ConfirmEntries), 1, 100000},
		{"confirm_bytes", p.ConfirmBytes, 1, 1 << 50},
	}
	for _, c := range checks {
		if c.value < c.min || c.value > c.max {
			return fmt.Errorf("%s must be from %d to %d, not %d", c.key, c.min, c.max, c.value)
		}
	}
	return nil
}

// PreferencesPath is the file of the preferences:
// <user config directory>/gh-sweep/preferences.json.
func PreferencesPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gh-sweep", "preferences.json"), nil
}

// LoadPreferences reads the file. A missing file gives the defaults. A key
// the file does not name keeps its default. An unknown key, bad JSON, and a
// value out of range are an error that names the file.
func LoadPreferences(path string) (Preferences, error) {
	p := DefaultPreferences()
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return DefaultPreferences(), fmt.Errorf("%s: %v", path, err)
	}
	if err := p.Check(); err != nil {
		return DefaultPreferences(), fmt.Errorf("%s: %v", path, err)
	}
	return p, nil
}

// SavePreferences writes the file with every key, indented by two spaces,
// with mode 0600 in a directory with mode 0700.
func SavePreferences(path string, p Preferences) error {
	if err := p.Check(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}
