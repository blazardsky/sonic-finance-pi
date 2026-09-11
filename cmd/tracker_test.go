package main

import (
	"net/http"
	"reflect"
	"testing"
)

// A Tracker group as the API hands it out: one per normalized (name,
// category), a last/min/max per Store, and a yearly average across every
// Store.
type trackerItemJSON struct {
	Name          string                   `json:"name"`
	CategoryID    int64                    `json:"category_id"`
	Stores        []trackerStoreJSON       `json:"stores"`
	YearlyAverage []trackerYearlyPriceJSON `json:"yearly_average"`
}

type trackerStoreJSON struct {
	Store          string  `json:"store"`
	LastPrice      float64 `json:"last_price"`
	LastOccurredOn string  `json:"last_occurred_on"`
	MinPrice       float64 `json:"min_price"`
	MaxPrice       float64 `json:"max_price"`
}

type trackerYearlyPriceJSON struct {
	Year         int     `json:"year"`
	AveragePrice float64 `json:"average_price"`
}

func (a *testApp) tracker(t *testing.T) []trackerItemJSON {
	t.Helper()
	var got []trackerItemJSON
	if res := a.get(t, "/api/tracker", &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/tracker = %d, want 200", res.StatusCode)
	}
	return got
}

// findTrackerGroup locates the group for a normalized name, or fails.
func findTrackerGroup(t *testing.T, got []trackerItemJSON, name string) trackerItemJSON {
	t.Helper()
	for _, g := range got {
		if g.Name == name {
			return g
		}
	}
	t.Fatalf("no tracker group named %q in %+v", name, got)
	return trackerItemJSON{}
}

func findTrackerStore(t *testing.T, g trackerItemJSON, store string) trackerStoreJSON {
	t.Helper()
	for _, s := range g.Stores {
		if s.Store == store {
			return s
		}
	}
	t.Fatalf("no store %q in tracker group %+v", store, g)
	return trackerStoreJSON{}
}

// Items with the same name and Store spelled differently — trimmed, cased
// and spaced differently — land in one group, per ADR-0013.
func TestTrackerGroupsByNormalizedNameAndStore(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID

	lt := "lt"
	one := 1.0
	a.addExpense(t, map[string]any{
		"occurred_on": "2025-06-10", "amount_cents": 500, "category_id": alimentari, "store": "Esselunga",
		"items": []map[string]any{{"name": "Latte", "amount_cents": 120, "category_id": alimentari, "quantity": one, "unit": lt}},
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2025-11-20", "amount_cents": 500, "category_id": alimentari, "store": "esselunga ",
		"items": []map[string]any{{"name": " LATTE", "amount_cents": 150, "category_id": alimentari, "quantity": one, "unit": lt}},
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-05", "amount_cents": 500, "category_id": alimentari, "store": "Conad",
		"items": []map[string]any{{"name": "latte", "amount_cents": 100, "category_id": alimentari, "quantity": one, "unit": lt}},
	})

	got := a.tracker(t)
	g := findTrackerGroup(t, got, "latte")
	if g.CategoryID != alimentari {
		t.Errorf("category_id = %d, want %d", g.CategoryID, alimentari)
	}
	if len(g.Stores) != 2 {
		t.Fatalf("stores = %+v, want 2 (esselunga, conad)", g.Stores)
	}

	esselunga := findTrackerStore(t, g, "esselunga")
	if esselunga.LastPrice != 150 || esselunga.LastOccurredOn != "2025-11-20" {
		t.Errorf("esselunga last = %v on %q, want 150 on 2025-11-20", esselunga.LastPrice, esselunga.LastOccurredOn)
	}
	if esselunga.MinPrice != 120 || esselunga.MaxPrice != 150 {
		t.Errorf("esselunga min/max = %v/%v, want 120/150", esselunga.MinPrice, esselunga.MaxPrice)
	}

	conad := findTrackerStore(t, g, "conad")
	if conad.LastPrice != 100 || conad.MinPrice != 100 || conad.MaxPrice != 100 {
		t.Errorf("conad = %+v, want last/min/max all 100", conad)
	}

	// Yearly average is across every Store: 2025 folds both Esselunga
	// purchases (120, 150), 2026 has just the one Conad purchase.
	wantYears := map[int]float64{2025: 135, 2026: 100}
	if len(g.YearlyAverage) != len(wantYears) {
		t.Fatalf("yearly_average = %+v, want one entry per year in %v", g.YearlyAverage, wantYears)
	}
	for _, y := range g.YearlyAverage {
		if want, ok := wantYears[y.Year]; !ok || y.AveragePrice != want {
			t.Errorf("yearly_average[%d] = %v, want %v", y.Year, y.AveragePrice, want)
		}
	}
}

// An Item with no quantity/unit has no price per unit to compare, so the
// Tracker falls back to its paid amount_cents.
func TestTrackerFallsBackToAmountCentsWithoutAQuantity(t *testing.T) {
	a := newTestApp(t)
	casa := a.category(t, "Casa").ID

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-02-01", "amount_cents": 5000, "category_id": casa, "store": "Ikea",
		"items": []map[string]any{{"name": "Lampada", "amount_cents": 1999, "category_id": casa}},
	})

	got := a.tracker(t)
	g := findTrackerGroup(t, got, "lampada")
	s := findTrackerStore(t, g, "ikea")
	if s.LastPrice != 1999 || s.MinPrice != 1999 || s.MaxPrice != 1999 {
		t.Errorf("ikea = %+v, want last/min/max all 1999 (amount_cents, no quantity)", s)
	}
}

// The same Item name under two different Categories is two groups: Category
// is part of the grouping key, same as Store.
func TestTrackerSeparatesGroupsByCategory(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID
	casa := a.category(t, "Casa").ID

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-10", "amount_cents": 500, "category_id": alimentari, "store": "Conad",
		"items": []map[string]any{{"name": "Sacchetti", "amount_cents": 50, "category_id": alimentari}},
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-11", "amount_cents": 500, "category_id": casa, "store": "Conad",
		"items": []map[string]any{{"name": "Sacchetti", "amount_cents": 300, "category_id": casa}},
	})

	got := a.tracker(t)
	var matches int
	for _, g := range got {
		if g.Name == "sacchetti" {
			matches++
		}
	}
	if matches != 2 {
		t.Fatalf("found %d groups named sacchetti, want 2 (one per category)", matches)
	}
}

// The whole point of the ticket: the Tracker is purely computed. Calling it
// must not add, remove or change a single Expense.
func TestTrackerWritesNothingToTheDatabase(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-10", "amount_cents": 500, "category_id": alimentari, "store": "Conad",
		"items": []map[string]any{{"name": "Pane", "amount_cents": 200, "category_id": alimentari}},
	})

	before := a.expenses(t)
	a.tracker(t)
	after := a.expenses(t)

	if !reflect.DeepEqual(before, after) {
		t.Errorf("expenses changed after GET /api/tracker: before = %+v, after = %+v", before, after)
	}
}
