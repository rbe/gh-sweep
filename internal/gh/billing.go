package gh

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rbe/gh-sweep/internal/project"
)

const usageJQ = `.usageItems[]?
| [
    (.product // ""),
    (.sku // ""),
    (.unitType // ""),
    ((.grossQuantity // 0) | tostring),
    ((.grossAmount // 0) | tostring),
    ((.discountAmount // 0) | tostring),
    ((.netAmount // 0) | tostring)
  ]
| join("\u001f")`

// Login reads the login of the logged-in user.
func Login() (string, error) {
	out, err := runGHFunc("api", "user", "--jq", ".login")
	if err != nil {
		return "", err
	}
	login := strings.TrimSpace(out)
	if login == "" {
		return "", fmt.Errorf("gh gave no login")
	}
	return login, nil
}

// BillingUsage reads the billing usage summary of the product in this
// month, for the user or, when org is not empty, for the organization.
func BillingUsage(user, org, product string) ([]project.UsageItem, error) {
	path := fmt.Sprintf("users/%s/settings/billing/usage/summary?product=%s", user, product)
	if org != "" {
		path = fmt.Sprintf("organizations/%s/settings/billing/usage/summary?product=%s", org, product)
	}
	out, err := runGHFunc("api", "-H", "Accept: application/vnd.github+json", path, "--jq", usageJQ)
	if err != nil {
		return nil, err
	}
	var items []project.UsageItem
	for _, f := range fieldLines(out, 7) {
		items = append(items, project.UsageItem{
			Product: f[0], SKU: f[1], Unit: f[2],
			Quantity: number(f[3]), Gross: number(f[4]), Discount: number(f[5]), Net: number(f[6]),
		})
	}
	return items, nil
}

// number reads a JSON number, and 0 for a text that is no number.
func number(s string) float64 {
	n, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return n
}
