package main

import (
	"net/http"
	"testing"
)

// Ticket 03: typeahead over historical Item names and Store values, backed by
// FTS5 (ADR-0015). Seeded through the ordinary Expense-creation path — these
// tables have no write surface of their own — then queried by prefix and by a
// small typo, the way a household member actually types.

func TestItemNameSuggestionsAreTypoTolerantAndRanked(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-01", "amount_cents": 200, "category_id": alimentari,
		"items": []map[string]any{{"name": "Melanzane", "amount_cents": 200, "category_id": alimentari}},
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-02", "amount_cents": 150, "category_id": alimentari,
		"items": []map[string]any{{"name": "Mele", "amount_cents": 150, "category_id": alimentari}},
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-03", "amount_cents": 300, "category_id": alimentari,
		"items": []map[string]any{{"name": "Pane", "amount_cents": 300, "category_id": alimentari}},
	})

	// A prefix: both "Melanzane" and "Mele" start with it, "Pane" does not.
	var prefix []string
	if res := a.get(t, "/api/items/suggest?q=mel", &prefix); res.StatusCode != http.StatusOK {
		t.Fatalf("GET suggest?q=mel = %d, want 200", res.StatusCode)
	}
	if !containsAll(prefix, "Melanzane", "Mele") || contains(prefix, "Pane") {
		t.Errorf("suggestions for %q = %v, want Melanzane and Mele, not Pane", "mel", prefix)
	}

	// A typo: "melanzana" for "Melanzane" should still come back, ranked
	// above the weaker partial match "Mele".
	var typo []string
	if res := a.get(t, "/api/items/suggest?q=melanzana", &typo); res.StatusCode != http.StatusOK {
		t.Fatalf("GET suggest?q=melanzana = %d, want 200", res.StatusCode)
	}
	if len(typo) == 0 || typo[0] != "Melanzane" {
		t.Errorf("suggestions for the typo %q = %v, want Melanzane ranked first", "melanzana", typo)
	}
}

func TestStoreSuggestionsAreTypoTolerantAndCaseInsensitive(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-01", "amount_cents": 200, "category_id": alimentari,
		"store": "Esselunga",
	})
	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-02", "amount_cents": 200, "category_id": alimentari,
		"store": "Coop",
	})

	var got []string
	if res := a.get(t, "/api/stores/suggest?q=esselunga", &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET stores/suggest?q=esselunga = %d, want 200", res.StatusCode)
	}
	if len(got) == 0 || got[0] != "Esselunga" {
		t.Errorf("store suggestions for %q = %v, want Esselunga first", "esselunga", got)
	}

	// A typo, lowercased: still finds it, case differences and a dropped
	// letter both tolerated by the trigram index.
	got = nil
	if res := a.get(t, "/api/stores/suggest?q=esseluga", &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET stores/suggest?q=esseluga = %d, want 200", res.StatusCode)
	}
	if !contains(got, "Esselunga") {
		t.Errorf("store suggestions for the typo %q = %v, want Esselunga", "esseluga", got)
	}
}

func TestSuggestEndpointsIgnoreAVeryShortQuery(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-01", "amount_cents": 200, "category_id": alimentari,
		"items": []map[string]any{{"name": "Pane", "amount_cents": 200, "category_id": alimentari}},
	})

	var got []string
	if res := a.get(t, "/api/items/suggest?q=", &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET suggest?q= = %d, want 200", res.StatusCode)
	}
	if len(got) != 0 {
		t.Errorf("suggestions for an empty query = %v, want none", got)
	}
}

// A literal double-quote in the query used to reach FTS5 as a Go-escaped
// (backslash) phrase instead of FTS5's own doubled-quote escaping, producing
// a malformed MATCH query. It should just find nothing, not fail the request.
func TestSuggestEndpointsToleratesAQuoteInTheQuery(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID

	a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-01", "amount_cents": 200, "category_id": alimentari,
		"items": []map[string]any{{"name": "Melanzane", "amount_cents": 200, "category_id": alimentari}},
	})

	var got []string
	res := a.get(t, `/api/items/suggest?q=mela"nzane`, &got)
	if res.StatusCode != http.StatusOK {
		t.Fatalf(`GET suggest?q=mela"nzane = %d, want 200`, res.StatusCode)
	}
}

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

func containsAll(list []string, want ...string) bool {
	for _, w := range want {
		if !contains(list, w) {
			return false
		}
	}
	return true
}
