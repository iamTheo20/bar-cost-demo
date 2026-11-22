package invoices

import (
	"reflect"
	"strings"
	"testing"

	"github.com/iamTheo20/bar-cost-demo/internal/core/decimal"
)

const dataDir = "../../../data/invoices/"

func mustLoad(t *testing.T, name string) Invoice {
	t.Helper()
	inv, err := Load(dataDir+name, nil)
	if err != nil {
		t.Fatal(err)
	}
	return inv
}

func TestAnInvoiceIsReadIntoTypedLines(t *testing.T) {
	inv := mustLoad(t, "NB-48211.txt")
	if inv.Supplier != "Nordic Beverage AB" || len(inv.Lines) != 4 {
		t.Fatalf("got %+v", inv)
	}
	if inv.Lines[1].UnitPrice.Cmp(decimal.MustParse("12.90")) != 0 || inv.Lines[1].UnitPrice.String() != "12.90" {
		t.Errorf("unit price %s", inv.Lines[1].UnitPrice)
	}
	if inv.Subtotal.String() != "5823.20" {
		t.Errorf("subtotal %s", inv.Subtotal)
	}
}

func TestArithmeticIssues(t *testing.T) {
	tests := []struct {
		file string
		want []string
	}{
		{"SF-1093.txt", []string{"Cheddar: 2 x 145.00 = 290.00, invoiced 300.00"}},
		{"NB-48211.txt", nil},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			if got := ArithmeticIssues(mustLoad(t, tt.file)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestASubtotalThatDoesNotAddUpIsReported(t *testing.T) {
	inv, err := RuleBasedExtractor{}.Parse("SUPPLIER: A\nINVOICE: 1\nDATE: 2026-11-01\nLime | kg | 2 | 2.00 | 4.00\nSUBTOTAL: 5,00\n")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"subtotal: lines add up to 4.00, invoiced 5.00"}
	if got := ArithmeticIssues(inv); !reflect.DeepEqual(got, want) {
		t.Errorf("got %q", got)
	}
}

func TestParseRefusesBrokenInvoices(t *testing.T) {
	tests := []struct {
		name, text, wantErr string
	}{
		{"no headers", "ITEM | UNIT | QTY | UNIT PRICE | TOTAL\nLime | kg | 1 | 2 | 2\n", "missing SUPPLIER, INVOICE, DATE, SUBTOTAL"},
		{"non-number names its product", "SUPPLIER: A\nINVOICE: 1\nDATE: 2026-11-01\nLime | kg | two | 2.00 | 4.00\nSUBTOTAL: 4\n", "Lime qty is not a number: 'two'"},
		{"no lines", "SUPPLIER: A\nINVOICE: 1\nDATE: 2026-11-01\nSUBTOTAL: 4\n", "no line items"},
		{"bad date", "SUPPLIER: A\nINVOICE: 1\nDATE: 1 Nov\nLime | kg | 1 | 2 | 2\nSUBTOTAL: 2\n", "invalid isoformat string: '1 Nov'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RuleBasedExtractor{}.Parse(tt.text)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("err = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseAmountReadsPrintedForms(t *testing.T) {
	tests := map[string]string{"1 234,50": "1234.50", " 12.90 ": "12.90", "-3": "-3"}
	for in, want := range tests {
		got, err := ParseAmount(in, "x")
		if err != nil || got.String() != want {
			t.Errorf("ParseAmount(%q) = %s, %v; want %s", in, got, err, want)
		}
	}
}

func TestPyRepr(t *testing.T) {
	tests := []struct {
		in   any
		want string
	}{
		{"gave up", "'gave up'"},
		{"it's", `"it's"`},
		{nil, "None"},
		{true, "True"},
		{"a\nb", `'a\nb'`},
	}
	for _, tt := range tests {
		if got := PyRepr(tt.in); got != tt.want {
			t.Errorf("PyRepr(%v) = %s, want %s", tt.in, got, tt.want)
		}
	}
}
