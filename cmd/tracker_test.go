package main

import (
	"net/http"
	"reflect"
	"testing"
)

// A Tracker group as the API hands it out: one per normalized (name,
// category), with every purchase of it oldest first.
type trackerItemJSON struct {
	Name       string             `json:"name"`
	CategoryID int64              `json:"category_id"`
	Points     []trackerPointJSON `json:"points"`
}

type trackerPointJSON struct {
	OccurredOn    string  `json:"occurred_on"`
	Price         float64 `json:"price"`
	Unit          string  `json:"unit"`
	Store         string  `json:"store"`
	Discounted    bool    `json:"discounted"`
	SubcategoryID *int64  `json:"subcategory_id"`
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

// Items with the same name spelled differently — trimmed, cased and spaced
// differently — land in one group (ADR-0013), with one point per purchase in
// date order, priced per unit, whatever order they were entered in.
func TestTrackerGroupsByNormalizedNameWithOnePointPerPurchase(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID
	bio := a.createSubcategoryNamed(t, "Bio")

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-01-05", "amount_cents": 500, "category_id": alimentari, "store": "Conad",
		"items": []map[string]any{{"name": "latte", "amount_cents": 200, "category_id": alimentari,
			"quantity": 2.0, "unit": "lt", "discounted": true, "subcategory_id": bio.ID}},
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2025-06-10", "amount_cents": 500, "category_id": alimentari, "store": " Esselunga",
		"items": []map[string]any{{"name": " LATTE", "amount_cents": 120, "category_id": alimentari,
			"quantity": 1.0, "unit": "lt"}},
	})

	g := findTrackerGroup(t, a.tracker(t), "latte")
	if g.CategoryID != alimentari {
		t.Errorf("category_id = %d, want %d", g.CategoryID, alimentari)
	}
	if len(g.Points) != 2 {
		t.Fatalf("points = %+v, want 2", g.Points)
	}
	first, second := g.Points[0], g.Points[1]
	if first.OccurredOn != "2025-06-10" || first.Price != 120 || first.Unit != "lt" ||
		first.Store != "Esselunga" || first.Discounted || first.SubcategoryID != nil {
		t.Errorf("first point = %+v, want 2025-06-10, 120/lt at Esselunga, not discounted, no tag", first)
	}
	if second.OccurredOn != "2026-01-05" || second.Price != 100 || second.Unit != "lt" ||
		second.Store != "Conad" || !second.Discounted ||
		second.SubcategoryID == nil || *second.SubcategoryID != bio.ID {
		t.Errorf("second point = %+v, want 2026-01-05, 100/lt at Conad, discounted, tag %d", second, bio.ID)
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

	g := findTrackerGroup(t, a.tracker(t), "lampada")
	if len(g.Points) != 1 || g.Points[0].Price != 1999 || g.Points[0].Unit != "" {
		t.Errorf("points = %+v, want one at 1999 with no unit (amount_cents, no quantity)", g.Points)
	}
}

// The same Item name under two different Categories is two groups: Category
// is part of the grouping key.
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
