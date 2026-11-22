package catalog

import (
	"errors"
	"testing"
	"time"

	"github.com/iamTheo20/bar-cost-demo/internal/core/decimal"
	"github.com/iamTheo20/bar-cost-demo/internal/core/invoices"
)

const data = "../../../data/"

func load(t *testing.T) (History, []MenuItem) {
	t.Helper()
	h, err := PriceHistory(data + "price_history.csv")
	if err != nil {
		t.Fatal(err)
	}
	m, err := Menu(data+"menu.csv", data+"recipes.csv")
	if err != nil {
		t.Fatal(err)
	}
	return h, m
}

func item(t *testing.T, items []MenuItem, name string) MenuItem {
	t.Helper()
	for _, m := range items {
		if m.Name == name {
			return m
		}
	}
	t.Fatalf("no menu item %s", name)
	return MenuItem{}
}

func TestLastPriceBefore(t *testing.T) {
	h, _ := load(t)
	tests := []struct {
		day     time.Time
		want    string
		wantNil bool
	}{
		{time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC), "64.00", false},
		{time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC), "61.00", false},
		{time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), "", true},
	}
	for _, tt := range tests {
		t.Run(tt.day.Format(invoices.DateLayout), func(t *testing.T) {
			p, ok := LastPriceBefore(h, "Avocado", tt.day)
			if ok == tt.wantNil || (ok && p.UnitPrice.String() != tt.want) {
				t.Errorf("got %v %v", p, ok)
			}
		})
	}
}

func TestTheInvoiceRaisesTheServingCost(t *testing.T) {
	h, items := load(t)
	inv, err := invoices.Load(data+"invoices/NB-48211.txt", nil)
	if err != nil {
		t.Fatal(err)
	}
	gt := item(t, items, "Gin & Tonic")
	tests := []struct {
		name string
		inv  *invoices.Invoice
		want string
	}{
		{"history only", nil, "32.187900"},
		{"with invoice", &inv, "33.667900"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ServingCost(gt, CurrentPrices(h, tt.inv))
			if err != nil || got.String() != tt.want {
				t.Errorf("got %s, %v; want %s", got, err, tt.want)
			}
		})
	}
}

func TestAnIngredientWithNoPriceIsAnErrorNotAZero(t *testing.T) {
	_, items := load(t)
	burger := item(t, items, "Cheeseburger")
	_, err := ServingCost(burger, map[string]decimal.Decimal{
		"Beef mince": decimal.New(120, 0), "Burger bun": decimal.MustParse("6.20"),
	})
	var missing *MissingPriceError
	if !errors.As(err, &missing) || missing.Product != "Cheddar" {
		t.Errorf("err = %v", err)
	}
}

func TestMenuKeepsFileOrderAndRecipes(t *testing.T) {
	_, items := load(t)
	want := []string{"Gin & Tonic", "Pilsner 40cl", "Cheeseburger", "Guacamole & chips"}
	for i, name := range want {
		if items[i].Name != name {
			t.Fatalf("item %d = %s, want %s", i, items[i].Name, name)
		}
	}
	if len(items[0].Recipe) != 3 || items[0].Recipe[0].Product != "Herno Gin 70cl" {
		t.Errorf("recipe %+v", items[0].Recipe)
	}
}
