// Package format turns sizes, times, and Git refs into the text shown on screen.
package format

import (
	"fmt"
	"strings"
	"time"
)

// Now is the clock. Tests replace it.
var Now = time.Now

func HumanSize(b int64) string {
	if b < 0 {
		b = 0
	}
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	f := float64(b)
	i := 0
	for f >= 1024 && i < len(units)-1 {
		f /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d B", int64(f))
	}
	if f >= 100 {
		return fmt.Sprintf("%.0f %s", f, units[i])
	}
	return fmt.Sprintf("%.1f %s", f, units[i])
}

func HumanTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := Now().Sub(t)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d/time.Minute))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d/time.Hour))
	case d < 14*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d/(24*time.Hour)))
	default:
		return t.Local().Format("2006-01-02")
	}
}

// Date is the UTC calendar date of t, as 2026-11-28, or "-" for the zero time.
func Date(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.UTC().Format("2006-01-02")
}

func ParseGitHubTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" || s == "null" {
		return time.Time{}
	}
	if i := strings.IndexByte(s, '.'); i >= 0 {
		frac := s[i+1:]
		j := 0
		for j < len(frac) && frac[j] >= '0' && frac[j] <= '9' {
			j++
		}
		s = s[:i] + frac[j:]
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z07:00"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func ShortenRef(r string) string {
	switch {
	case r == "":
		return "-"
	case strings.HasPrefix(r, "refs/heads/"):
		return strings.TrimPrefix(r, "refs/heads/")
	case strings.HasPrefix(r, "refs/tags/"):
		return "tag:" + strings.TrimPrefix(r, "refs/tags/")
	case strings.HasPrefix(r, "refs/pull/") && strings.HasSuffix(r, "/merge"):
		n := strings.TrimSuffix(strings.TrimPrefix(r, "refs/pull/"), "/merge")
		return "pr/" + n
	case strings.HasPrefix(r, "refs/pull/") && strings.HasSuffix(r, "/head"):
		n := strings.TrimSuffix(strings.TrimPrefix(r, "refs/pull/"), "/head")
		return "pr/" + n
	default:
		return r
	}
}

func StripControls(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, s)
}

func Fit(s string, n int) string {
	r := []rune(s)
	if n <= 0 {
		return ""
	}
	if len(r) <= n {
		return s
	}
	if n == 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}

func PadRight(s string, n int) string {
	r := []rune(s)
	if len(r) >= n {
		return string(r[:n])
	}
	return s + strings.Repeat(" ", n-len(r))
}
