// Package invoice reads a supplier invoice and checks its arithmetic.
//
//	barcost invoice <file> [--summary] [--llm]
package invoice

import (
	"fmt"
	"strings"

	"github.com/iamTheo20/bar-cost-demo/internal/core/cli"
	"github.com/iamTheo20/bar-cost-demo/internal/core/invoices"
)

// Name is the feature's command name.
const Name = "invoice"

// Run is the invoice's lines, every arithmetic issue, and a one-line summary last.
func Run(args []string) (string, error) {
	parsed, err := cli.InvoiceArgsFromEnv(args)
	if err != nil {
		return "", err
	}
	inv, err := parsed.Load()
	if err != nil {
		return "", err
	}
	return Report(inv, parsed.SummaryOnly), nil
}

// Report renders an invoice check: the summary alone, or lines, issues and summary.
func Report(inv invoices.Invoice, summaryOnly bool) string {
	issues := invoices.ArithmeticIssues(inv)
	summary := fmt.Sprintf("invoice %s: %d lines, %s SEK, %d arithmetic issue(s)",
		inv.Number, len(inv.Lines), inv.Subtotal, len(issues))
	if summaryOnly {
		return summary
	}
	report := []string{fmt.Sprintf("%s, invoice %s, %s", inv.Supplier, inv.Number, inv.Date.Format(invoices.DateLayout))}
	for _, i := range inv.Lines {
		report = append(report, fmt.Sprintf("  %-24s %5s %-7s x %8s = %9s",
			i.Product, i.Qty, i.Unit, i.UnitPrice, i.Total))
	}
	for _, issue := range issues {
		report = append(report, "  ISSUE "+issue)
	}
	return strings.Join(append(report, summary), "\n")
}
