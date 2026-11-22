// Package prices flags price moves against the last purchase.
//
//	barcost prices <file> [--summary] [--llm]
package prices

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/iamTheo20/bar-cost-demo/internal/core/catalog"
	"github.com/iamTheo20/bar-cost-demo/internal/core/cli"
	"github.com/iamTheo20/bar-cost-demo/internal/core/decimal"
	"github.com/iamTheo20/bar-cost-demo/internal/core/invoices"
)

// Name is the feature's command name.
const Name = "prices"

// Threshold is the relative move, up or down, that gets flagged.
var Threshold = decimal.MustParse("0.05")

// Run is each line's change since the last purchase, flagged at 5% or
// more, and a summary last.
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
	return Report(inv, history, parsed.SummaryOnly)
}

type move struct {
	product string
	change  decimal.Decimal
}

// Report renders every line's move against history: the summary alone, or
// one row per line and the summary.
func Report(inv invoices.Invoice, history catalog.History, summaryOnly bool) (string, error) {
	report := []string{fmt.Sprintf("Price changes on %s vs the last purchase:", inv.Number)}
	var moved []move
	for _, item := range inv.Lines {
		last, ok := catalog.LastPriceBefore(history, item.Product, inv.Date)
		if !ok {
			report = append(report, fmt.Sprintf("  %-24s %8s -> %8s   no history", item.Product, "new", item.UnitPrice))
			continue
		}
		if last.UnitPrice.IsZero() {
			return "", fmt.Errorf("the last price of %q is 0, so its change cannot be computed", item.Product)
		}
		change := item.UnitPrice.Sub(last.UnitPrice).Quo(last.UnitPrice)
		mark := ""
		if change.Abs().Cmp(Threshold) >= 0 {
			moved = append(moved, move{item.Product, change})
			mark = "  FLAG"
		}
		report = append(report, fmt.Sprintf("  %-24s %8s -> %8s  %7s%s",
			item.Product, last.UnitPrice, item.UnitPrice, change.Percent(1, true), mark))
	}
	summary := fmt.Sprintf("prices %s: %d of %d items moved 5%% or more", inv.Number, len(moved), len(inv.Lines))
	if len(moved) > 0 {
		largest := moved[0]
		for _, m := range moved[1:] {
			if m.change.Abs().Cmp(largest.change.Abs()) > 0 {
				largest = m
			}
		}
		summary += fmt.Sprintf(" (largest: %s %s)", largest.product, largest.change.Percent(1, true))
	}
	if summaryOnly {
		return summary, nil
	}
	return strings.Join(append(report, summary), "\n"), nil
}
