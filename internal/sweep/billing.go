package sweep

import (
	"fmt"
	"time"

	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
)

var billingAction = Action{
	Flag: "--billing", Title: "Show the billed Actions usage of this month", Verb: "print", Noun: "billed Actions usage",
	ReadOnly: true, TotalLine: project.BillingTotalLine, Account: billingEntries, Check: checkBillingScope,
}

// checkBillingScope refuses repository arguments, because GitHub bills an
// account, not a repository.
func checkBillingScope(s Scope, _ project.Filter) error {
	if len(s.Repos) > 0 {
		return fmt.Errorf("--billing reads an account. Pass --user NAME, --org NAME, or no target for yourself")
	}
	return nil
}

// billingEntries reads the Actions items of the usage summary of the user,
// the organization, or the logged-in user.
func billingEntries(s Scope, _ project.Filter, _ time.Time) ([]Entry, error) {
	account := s.Org
	if account == "" {
		account = s.User
	}
	if account == "" {
		login, err := gh.Login()
		if err != nil {
			return nil, err
		}
		account = login
	}
	user := account
	if s.Org != "" {
		user = ""
	}
	items, err := gh.BillingUsage(user, s.Org, "Actions")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", account, err)
	}
	var entries []Entry
	for _, it := range items {
		line := project.UsageItemLine(account, it)
		entries = append(entries, Entry{Line: func(bool) string { return line }, Total: project.Total{Count: 1, Gross: it.Gross, Net: it.Net}})
	}
	return entries, nil
}
