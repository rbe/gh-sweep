package project

import "fmt"

// UsageItem is one line of the billing usage summary of GitHub: a product,
// a SKU, the quantity in its unit, and the amounts in US dollars.
type UsageItem struct {
	Product  string
	SKU      string
	Unit     string
	Quantity float64
	Gross    float64
	Discount float64
	Net      float64
}

// UsageItemLine is the line of one usage item after --billing.
func UsageItemLine(account string, u UsageItem) string {
	return fmt.Sprintf("%s: %s %s, %g %s, gross $%.2f, discount $%.2f, net $%.2f",
		account, u.Product, u.SKU, u.Quantity, u.Unit, u.Gross, u.Discount, u.Net)
}

// BillingTotalLine is the last line of --billing.
func BillingTotalLine(t Total, _ bool) string {
	return fmt.Sprintf("total: items %d, gross $%.2f, net $%.2f", t.Count, t.Gross, t.Net)
}
