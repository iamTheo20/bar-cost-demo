// Package catalog is what the venue knows besides invoices: past prices,
// recipes and the menu.
package catalog

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/iamTheo20/bar-cost-demo/internal/core/decimal"
	"github.com/iamTheo20/bar-cost-demo/internal/core/invoices"
)

// PricePoint is one past purchase price for a product.
type PricePoint struct {
	Date      time.Time
	Supplier  string
	UnitPrice decimal.Decimal
}

// Ingredient is how much of one product a serving uses.
type Ingredient struct {
	Product string
	Qty     decimal.Decimal
}

// MenuItem is a dish or drink: its price excluding VAT, the margin it
// should keep and the ingredients of one serving.
type MenuItem struct {
	Name         string
	Price        decimal.Decimal
	TargetMargin decimal.Decimal
	Recipe       []Ingredient
}

// History is each product's past prices, oldest first.
type History map[string][]PricePoint

// rows reads a CSV with a header row into one map per record.
func rows(path string) ([]map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(records) == 0 {
		return nil, nil
	}
	head := records[0]
	if len(head) > 0 {
		head[0] = trimBOM(head[0])
	}
	out := make([]map[string]string, 0, len(records)-1)
	for _, rec := range records[1:] {
		row := make(map[string]string, len(head))
		for i, key := range head {
			if i < len(rec) {
				row[key] = rec[i]
			}
		}
		out = append(out, row)
	}
	return out, nil
}

func trimBOM(s string) string {
	if len(s) >= 3 && s[:3] == "\xef\xbb\xbf" {
		return s[3:]
	}
	return s
}

func field(row map[string]string, key, path string) (string, error) {
	v, ok := row[key]
	if !ok {
		return "", fmt.Errorf("%s: a row has no %s column", path, key)
	}
	return v, nil
}

func number(row map[string]string, key, path string) (decimal.Decimal, error) {
	raw, err := field(row, key, path)
	if err != nil {
		return decimal.Decimal{}, err
	}
	d, err := decimal.Parse(raw)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("%s: %s %w", path, key, err)
	}
	return d, nil
}

// PriceHistory reads price_history.csv: each product's past prices, oldest first.
func PriceHistory(path string) (History, error) {
	records, err := rows(path)
	if err != nil {
		return nil, err
	}
	history := History{}
	for _, row := range records {
		rawDate, err := field(row, "date", path)
		if err != nil {
			return nil, err
		}
		date, err := time.Parse(invoices.DateLayout, rawDate)
		if err != nil {
			return nil, fmt.Errorf("%s: invalid date %q", path, rawDate)
		}
		price, err := number(row, "unit_price", path)
		if err != nil {
			return nil, err
		}
		product, err := field(row, "product", path)
		if err != nil {
			return nil, err
		}
		history[product] = append(history[product], PricePoint{date, row["supplier"], price})
	}
	for _, points := range history {
		sort.SliceStable(points, func(i, j int) bool { return points[i].Date.Before(points[j].Date) })
	}
	return history, nil
}

// LastPriceBefore is the most recent purchase of product strictly before day.
func LastPriceBefore(h History, product string, day time.Time) (PricePoint, bool) {
	var last PricePoint
	found := false
	for _, p := range h[product] {
		if p.Date.Before(day) {
			last, found = p, true
		}
	}
	return last, found
}

// Menu reads menu.csv and recipes.csv: the menu, in file order, each item
// with the ingredients one serving uses.
func Menu(menuPath, recipesPath string) ([]MenuItem, error) {
	recipeRows, err := rows(recipesPath)
	if err != nil {
		return nil, err
	}
	recipes := map[string][]Ingredient{}
	for _, row := range recipeRows {
		qty, err := number(row, "qty", recipesPath)
		if err != nil {
			return nil, err
		}
		name, err := field(row, "menu_item", recipesPath)
		if err != nil {
			return nil, err
		}
		recipes[name] = append(recipes[name], Ingredient{row["product"], qty})
	}
	menuRows, err := rows(menuPath)
	if err != nil {
		return nil, err
	}
	items := make([]MenuItem, 0, len(menuRows))
	for _, row := range menuRows {
		price, err := number(row, "price", menuPath)
		if err != nil {
			return nil, err
		}
		target, err := number(row, "target_margin", menuPath)
		if err != nil {
			return nil, err
		}
		name := row["menu_item"]
		items = append(items, MenuItem{name, price, target, recipes[name]})
	}
	return items, nil
}

// CurrentPrices is the latest known price per product; an invoice's prices
// replace the history's. inv may be nil.
func CurrentPrices(h History, inv *invoices.Invoice) map[string]decimal.Decimal {
	prices := map[string]decimal.Decimal{}
	for product, points := range h {
		if len(points) > 0 {
			prices[product] = points[len(points)-1].UnitPrice
		}
	}
	if inv != nil {
		for _, item := range inv.Lines {
			prices[item.Product] = item.UnitPrice
		}
	}
	return prices
}

// MissingPriceError names an ingredient with no known price.
type MissingPriceError struct{ Product string }

func (e *MissingPriceError) Error() string {
	return fmt.Sprintf("no known price for %q", e.Product)
}

// ServingCost is what one serving costs in ingredients. A product with no
// known price is a *MissingPriceError, never a zero.
func ServingCost(item MenuItem, prices map[string]decimal.Decimal) (decimal.Decimal, error) {
	cost := decimal.New(0, 0)
	for _, ing := range item.Recipe {
		price, ok := prices[ing.Product]
		if !ok {
			return decimal.Decimal{}, &MissingPriceError{ing.Product}
		}
		cost = cost.Add(ing.Qty.Mul(price))
	}
	return cost, nil
}
