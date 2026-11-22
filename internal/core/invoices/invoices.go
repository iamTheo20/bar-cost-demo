// Package invoices is supplier invoices: one data model, one way to read them.
//
// An Extractor turns an invoice file into an Invoice. RuleBasedExtractor reads
// the plain-text layout in data/invoices/; the LLM extractor in
// core/llmextract reads PDFs and photos behind the same interface, and
// nothing downstream changes.
package invoices

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/iamTheo20/bar-cost-demo/internal/core/decimal"
)

// DateLayout is the only date form invoices are read in: ISO YYYY-MM-DD.
const DateLayout = "2006-01-02"

// LineItem is one invoice row: what was bought, how much, at what price.
type LineItem struct {
	Product   string
	Unit      string
	Qty       decimal.Decimal
	UnitPrice decimal.Decimal
	Total     decimal.Decimal
}

// Invoice is a supplier invoice, amounts in SEK.
type Invoice struct {
	Supplier string
	Number   string
	Date     time.Time
	Lines    []LineItem
	Subtotal decimal.Decimal
}

// Extractor turns an invoice file into an Invoice, or says why it cannot.
type Extractor interface {
	Extract(path string) (Invoice, error)
}

// ParseAmount reads an amount as printed: spaces dropped, a decimal comma
// read as a point. what names the field in the error.
func ParseAmount(raw, what string) (decimal.Decimal, error) {
	clean := strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(raw), " ", ""), ",", ".")
	d, err := decimal.Parse(clean)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("%s is not a number: %s", what, pyQuote(raw))
	}
	return d, nil
}

// isUpper is Python's str.isupper: at least one cased letter, none lower.
func isUpper(s string) bool {
	cased := false
	for _, r := range s {
		if unicode.IsLower(r) || unicode.IsTitle(r) {
			return false
		}
		cased = cased || unicode.IsUpper(r)
	}
	return cased
}

func headers(lines []string) map[string]string {
	found := map[string]string{}
	for _, line := range lines {
		key, value, ok := strings.Cut(line, ":")
		if ok && isUpper(strings.TrimSpace(key)) && !strings.Contains(line, "|") {
			found[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return found
}

func rows(lines []string) ([]LineItem, error) {
	var items []LineItem
	for _, line := range lines {
		cells := strings.Split(line, "|")
		if len(cells) != 5 {
			continue
		}
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		if strings.ToUpper(cells[0]) == "ITEM" {
			continue
		}
		product := cells[0]
		qty, err := ParseAmount(cells[2], product+" qty")
		if err != nil {
			return nil, err
		}
		price, err := ParseAmount(cells[3], product+" unit price")
		if err != nil {
			return nil, err
		}
		total, err := ParseAmount(cells[4], product+" total")
		if err != nil {
			return nil, err
		}
		items = append(items, LineItem{product, cells[1], qty, price, total})
	}
	return items, nil
}

// RuleBasedExtractor reads the demo's plain-text layout: headers, a pipe
// table, a subtotal.
type RuleBasedExtractor struct{}

// Extract reads and parses one plain-text invoice file.
func (RuleBasedExtractor) Extract(path string) (Invoice, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Invoice{}, err
	}
	return RuleBasedExtractor{}.Parse(string(b))
}

// Parse parses one invoice, refusing anything with a header or the table
// missing.
func (RuleBasedExtractor) Parse(text string) (Invoice, error) {
	var lines []string
	for _, line := range splitLines(text) {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	head := headers(lines)
	var missing []string
	for _, k := range []string{"SUPPLIER", "INVOICE", "DATE", "SUBTOTAL"} {
		if _, ok := head[k]; !ok {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return Invoice{}, fmt.Errorf("invoice is missing %s", strings.Join(missing, ", "))
	}
	items, err := rows(lines)
	if err != nil {
		return Invoice{}, err
	}
	if len(items) == 0 {
		return Invoice{}, errors.New("invoice has no line items")
	}
	date, err := time.Parse(DateLayout, head["DATE"])
	if err != nil {
		return Invoice{}, fmt.Errorf("invalid isoformat string: %s", pyQuote(head["DATE"]))
	}
	subtotal, err := ParseAmount(head["SUBTOTAL"], "subtotal")
	if err != nil {
		return Invoice{}, err
	}
	return Invoice{head["SUPPLIER"], head["INVOICE"], date, items, subtotal}, nil
}

func splitLines(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.Split(strings.ReplaceAll(text, "\r", "\n"), "\n")
}

// Load is the invoice at path, read by extractor (the rule-based one when
// extractor is nil).
func Load(path string, extractor Extractor) (Invoice, error) {
	if extractor == nil {
		extractor = RuleBasedExtractor{}
	}
	return extractor.Extract(path)
}

// ArithmeticIssues is every place the supplier's own numbers do not add up.
func ArithmeticIssues(inv Invoice) []string {
	var issues []string
	sum := decimal.New(0, 0)
	for _, item := range inv.Lines {
		expected := item.Qty.Mul(item.UnitPrice).Quantize(-2)
		if expected.Cmp(item.Total) != 0 {
			issues = append(issues, fmt.Sprintf("%s: %s x %s = %s, invoiced %s",
				item.Product, item.Qty, item.UnitPrice, expected, item.Total))
		}
		sum = sum.Add(item.Total)
	}
	if sum.Cmp(inv.Subtotal) != 0 {
		issues = append(issues, fmt.Sprintf("subtotal: lines add up to %s, invoiced %s", sum, inv.Subtotal))
	}
	return issues
}

// pyQuote quotes s the way Python's repr quotes a string, so error messages
// read as they did in the Python version.
func pyQuote(s string) string { return PyRepr(s) }

// PyRepr renders a JSON-decoded value the way Python's repr would: strings
// in single quotes, nil as None, booleans as True/False.
func PyRepr(v any) string {
	switch x := v.(type) {
	case nil:
		return "None"
	case bool:
		if x {
			return "True"
		}
		return "False"
	case string:
		quote := "'"
		if strings.Contains(x, "'") && !strings.Contains(x, `"`) {
			quote = `"`
		}
		var b strings.Builder
		b.WriteString(quote)
		for _, r := range x {
			switch {
			case r == '\\':
				b.WriteString(`\\`)
			case string(r) == quote:
				b.WriteString(`\` + quote)
			case r == '\n':
				b.WriteString(`\n`)
			case r == '\r':
				b.WriteString(`\r`)
			case r == '\t':
				b.WriteString(`\t`)
			case !unicode.IsPrint(r) && r < 0x100:
				fmt.Fprintf(&b, `\x%02x`, r)
			default:
				b.WriteRune(r)
			}
		}
		b.WriteString(quote)
		return b.String()
	default:
		return fmt.Sprint(x)
	}
}
