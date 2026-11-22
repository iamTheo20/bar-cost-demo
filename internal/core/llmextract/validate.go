package llmextract

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/iamTheo20/bar-cost-demo/internal/core/decimal"
	"github.com/iamTheo20/bar-cost-demo/internal/core/invoices"
)

func object(v any, what string) (map[string]any, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s is not an object", what)
	}
	return m, nil
}

func text(data map[string]any, key string) (string, error) {
	s, ok := data[key].(string)
	if !ok || strings.TrimSpace(s) == "" {
		return "", fmt.Errorf("the extracted invoice has no usable %s", key)
	}
	return strings.TrimSpace(s), nil
}

func amount(data map[string]any, key string) (decimal.Decimal, error) {
	raw, err := text(data, key)
	if err != nil {
		return decimal.Decimal{}, err
	}
	d, err := decimal.Parse(raw)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("%s is not a number: %s", key, invoices.PyRepr(raw))
	}
	return d, nil
}

func line(raw any) (invoices.LineItem, error) {
	row, err := object(raw, "an invoice line")
	if err != nil {
		return invoices.LineItem{}, err
	}
	var item invoices.LineItem
	if item.Product, err = text(row, "product"); err != nil {
		return item, err
	}
	if item.Unit, err = text(row, "unit"); err != nil {
		return item, err
	}
	if item.Qty, err = amount(row, "qty"); err != nil {
		return item, err
	}
	if item.UnitPrice, err = amount(row, "unit_price"); err != nil {
		return item, err
	}
	item.Total, err = amount(row, "total")
	return item, err
}

// InvoiceFrom is an Invoice from Claude's structured output, every field
// checked, in the same order and with the same messages as the Python version.
func InvoiceFrom(structured any) (invoices.Invoice, error) {
	data, err := object(structured, "the extracted invoice")
	if err != nil {
		return invoices.Invoice{}, err
	}
	rawLines, ok := data["lines"].([]any)
	if !ok || len(rawLines) == 0 {
		return invoices.Invoice{}, errors.New("the extracted invoice has no line items")
	}
	inv := invoices.Invoice{}
	for _, raw := range rawLines {
		item, err := line(raw)
		if err != nil {
			return invoices.Invoice{}, err
		}
		inv.Lines = append(inv.Lines, item)
	}
	rawDate, err := text(data, "date")
	if err == nil {
		inv.Date, err = time.Parse(invoices.DateLayout, rawDate)
	}
	if err != nil {
		return invoices.Invoice{}, fmt.Errorf("date is not YYYY-MM-DD: %s", invoices.PyRepr(data["date"]))
	}
	if inv.Supplier, err = text(data, "supplier"); err != nil {
		return invoices.Invoice{}, err
	}
	if inv.Number, err = text(data, "number"); err != nil {
		return invoices.Invoice{}, err
	}
	if inv.Subtotal, err = amount(data, "subtotal"); err != nil {
		return invoices.Invoice{}, err
	}
	return inv, nil
}
