package main

import (
	"fmt"

	"github.com/rbe/gh-sweep/internal/gh"
	"github.com/rbe/gh-sweep/internal/project"
	"github.com/rbe/gh-sweep/internal/ui"
)

// billingJob is --billing: the billed GitHub Actions usage of this month.
var billingJob = job{
	flag: "--billing", verb: "print", noun: "billed Actions usage", readOnly: true,
	total: project.BillingTotalLine, account: billingUsage, target: checkBillingTarget,
}

// checkBillingTarget refuses repository arguments, because GitHub bills an
// account, not a repository.
func checkBillingTarget(o ui.Options, _ project.Filter) error {
	if len(o.Repos) > 0 {
		return fmt.Errorf("--billing reads an account. Pass --user NAME, --org NAME, or no target for yourself")
	}
	return nil
}

// billingUsage reads the Actions items of the usage summary of the user,
// the organization, or the logged-in user.
func billingUsage(o ui.Options, _ project.Filter, _ bool) ([]string, project.Total, error) {
	account := o.Org
	if account == "" {
		account = o.User
	}
	if account == "" {
		login, err := gh.Login()
		if err != nil {
			return nil, project.Total{}, err
		}
		account = login
	}
	user := account
	if o.Org != "" {
		user = ""
	}
	items, err := gh.BillingUsage(user, o.Org, "Actions")
	if err != nil {
		return nil, project.Total{}, fmt.Errorf("%s: %w", account, err)
	}
	var lines []string
	var t project.Total
	for _, it := range items {
		lines = append(lines, project.UsageItemLine(account, it))
		t = t.Add(project.Total{Count: 1, Gross: it.Gross, Net: it.Net})
	}
	return lines, t, nil
}
