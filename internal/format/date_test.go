// id: ADR-0005

package format

import (
	"testing"
	"time"
)

func TestDateIsTheUTCCalendarDateOrADash(t *testing.T) {
	late := time.Date(2026, 11, 28, 0, 30, 0, 0, time.FixedZone("CET", 3600))
	if Date(late) != "2026-11-27" || Date(time.Time{}) != "-" {
		t.Fatalf("%q %q", Date(late), Date(time.Time{}))
	}
}
