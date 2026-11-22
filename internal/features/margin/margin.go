// Package margin shows what an invoice's prices do to menu margins.
//
//	barcost margin <file> [--summary] [--llm]
package margin

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/iamTheo20/bar-cost-demo/internal/core/catalog"
	"github.com/iamTheo20/bar-cost-demo/internal/core/cli"
	"github.com/iamTheo20/bar-cost-demo/internal/core/decimal"
	"github.com/iamTheo20/bar-cost-demo/internal/core/invoices"
)

// Name is the feature's command name.
const Name = "margin"

// Run is the margin before and after for every menu item the invoice
// touches, and a summary last.
func Run(args []string) (string, error) {
	parsed, err := cli.InvoiceArgsFromEnv(args)
	if err != nil {
		return "", err
	}
	inv, err := parsed.Load()
	if err != nil {
		return "", err
	}
	history, err := catalog.PriceHistory(filepath.Join(parsed.DataDir, "price_history.csv"))
	if err != nil {
		return "", err
	}
	items, err := catalog.Menu(filepath.Join(parsed.DataDir, "menu.csv"), filepath.Join(parsed.DataDir, "recipes.csv"))
	if err != nil {
		return "", err
	}
	return Report(inv, history, items, parsed.SummaryOnly)
}

// Affected is the menu items, in menu order, whose recipe uses a product on the invoice.
func Affected(inv invoices.Invoice, items []catalog.MenuItem) []catalog.MenuItem {
	bought := map[string]bool{}
	for _, l := range inv.Lines {
		bought[l.Product] = true
	}
	var out []catalog.MenuItem
	for _, m := range items {
		if slices.ContainsFunc(m.Recipe, func(i catalog.Ingredient) bool { return bought[i.Product] }) {
			out = append(out, m)
		}
	}
	return out
}

// Margin is 1 - serving cost / price.
func Margin(item catalog.MenuItem, prices map[string]decimal.Decimal) (decimal.Decimal, error) {
	cost, err := catalog.ServingCost(item, prices)
	if err != nil {
		return decimal.Decimal{}, err
	}
	if item.Price.IsZero() {
		return decimal.Decimal{}, fmt.Errorf("menu item %q has price 0, so it has no margin", item.Name)
	}
	return decimal.New(1, 0).Sub(cost.Quo(item.Price)), nil
}

// Report renders margins before and after the invoice for every affected item.
func Report(inv invoices.Invoice, history catalog.History, items []catalog.MenuItem, summaryOnly bool) (string, error) {
	before, after := catalog.CurrentPrices(history, nil), catalog.CurrentPrices(history, &inv)
	affected := Affected(inv, items)
	report := []string{fmt.Sprintf("Menu margins after %s (price ex VAT, target in brackets):", inv.Number)}
	var below []string
	for _, item := range affected {
		old, err := Margin(item, before)
		if err != nil {
			return "", err
		}
		next, err := Margin(item, after)
		if err != nil {
			return "", err
		}
		target := item.TargetMargin.Percent(0, false)
		flag := ""
		if next.Cmp(item.TargetMargin) < 0 {
			below = append(below, fmt.Sprintf("%s %s < %s", item.Name, next.Percent(1, false), target))
			flag = "  BELOW TARGET"
		}
		report = append(report, fmt.Sprintf("  %-20s %6s -> %6s  (%s)%s",
			item.Name, old.Percent(1, false), next.Percent(1, false), target, flag))
	}
	summary := fmt.Sprintf("margin %s: %d of %d affected menu items below target", inv.Number, len(below), len(affected))
	if len(below) > 0 {
		summary += " (" + strings.Join(below, ", ") + ")"
	}
	if summaryOnly {
		return summary, nil
	}
	return strings.Join(append(report, summary), "\n"), nil
}
