// id: ADR-0006

package project

import "testing"

func TestSettingLinesSayUnchangedWouldSetOrSet(t *testing.T) {
	r := Retention{Days: 90, MaxDays: 400}
	for _, c := range []struct{ got, want string }{
		{RetentionLine("acme/widgets", r, 90, false), "acme/widgets: retention-days 90, unchanged (maximum 400)"},
		{RetentionLine("acme/widgets", r, 30, true), "acme/widgets: retention-days 90, would set 30 (maximum 400)"},
		{RetentionLine("acme/widgets", r, 30, false), "acme/widgets: retention-days 90, set 30 (maximum 400)"},
		{PRApprovalLine("acme/widgets", true, false, false), "acme/widgets: actions-pr-approval allow, set deny"},
		{PRApprovalLine("acme/widgets", false, false, true), "acme/widgets: actions-pr-approval deny, unchanged"},
	} {
		if c.got != c.want {
			t.Fatalf("got %q, want %q", c.got, c.want)
		}
	}
}

func TestCheckRetentionDaysAcceptsTheMaximumAndRefusesMore(t *testing.T) {
	r := Retention{Days: 90, MaxDays: 90}
	if err := CheckRetentionDays(90, r); err != nil {
		t.Fatal(err)
	}
	if err := CheckRetentionDays(91, r); err == nil {
		t.Fatal("91 days passed the check")
	}
}
