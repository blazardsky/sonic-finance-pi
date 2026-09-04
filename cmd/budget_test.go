package main

import (
	"net/http"
	"testing"
)

// Budget/Target/Goal as the API hands them out (ticket 04).
type budgetJSON struct {
	Available   bool  `json:"available"`
	BudgetCents int64 `json:"budget_cents"`
	TargetCents int64 `json:"target_cents"`
	GoalCents   int64 `json:"goal_cents"`
}

func (a *testApp) budget(t *testing.T) budgetJSON {
	t.Helper()
	var got budgetJSON
	if res := a.get(t, budgetPath, &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200", budgetPath, res.StatusCode)
	}
	return got
}

// testClock is 2026-03-15 (reports_test.go), so the current in-progress
// month is 2026-03 and the trailing completed months run back from 2026-02.

// Two completed months of history is exactly one short of the ticket's
// threshold; a third tips it into availability. This is the boundary the
// ticket names explicitly.
func TestBudgetUnavailableWithFewerThanThreeCompletedMonths(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")

	a.addExpense(t, map[string]any{"occurred_on": "2026-01-10", "amount_cents": 10000, "category_id": alimentari.ID})
	a.addExpense(t, map[string]any{"occurred_on": "2026-02-10", "amount_cents": 20000, "category_id": alimentari.ID})

	if got := a.budget(t); got.Available {
		t.Fatalf("available = true with 2 completed months of history, want false: %+v", got)
	}

	// A third completed month of history tips it over.
	a.addExpense(t, map[string]any{"occurred_on": "2025-12-10", "amount_cents": 30000, "category_id": alimentari.ID})

	if got := a.budget(t); !got.Available {
		t.Fatalf("available = false with 3 completed months of history, want true: %+v", got)
	}
}

// The whole ticket in one number: the median of a known, small set of
// completed months' Expense totals — not the average, which these totals
// were chosen to tell apart from the median.
func TestBudgetIsTheMedianOfTrailingCompletedMonths(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")

	// Four completed months (Nov, Dec, Jan, Feb), totals 4000/1000/3000/2000.
	// Sorted: 1000, 2000, 3000, 4000 — an even count, so the median is the
	// average of the two middle values: (2000+3000)/2 = 2500.
	for month, cents := range map[string]int64{
		"2025-11": 4000, "2025-12": 1000, "2026-01": 3000, "2026-02": 2000,
	} {
		a.addExpense(t, map[string]any{"occurred_on": month + "-10", "amount_cents": cents, "category_id": alimentari.ID})
	}

	got := a.budget(t)
	if !got.Available {
		t.Fatalf("available = false, want true: %+v", got)
	}
	if got.BudgetCents != 2500 {
		t.Errorf("budget_cents = %d, want 2500 (the median of the two middle values)", got.BudgetCents)
	}
}

// An odd count of months, where the median and the mean actually differ, so a
// budget that quietly computed an average instead would be caught.
func TestBudgetIsTheMedianNotTheMeanOverAnOddNumberOfMonths(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")

	// Three completed months: 100, 100, 100000. Mean = 33400; median = 100.
	for month, cents := range map[string]int64{
		"2025-12": 100, "2026-01": 100, "2026-02": 100000,
	} {
		a.addExpense(t, map[string]any{"occurred_on": month + "-10", "amount_cents": cents, "category_id": alimentari.ID})
	}

	got := a.budget(t)
	if !got.Available {
		t.Fatalf("available = false, want true: %+v", got)
	}
	if got.BudgetCents != 100 {
		t.Errorf("budget_cents = %d, want 100 (the median) — a mean would answer 33400", got.BudgetCents)
	}
}

// ADR-0009: Investments is excluded from Budget specifically, unlike every
// other report. An Investments-categorised Expense in a history month must
// not move that month's contribution to the median.
func TestBudgetExcludesTheInvestmentsCategory(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")
	investments := a.investments(t)

	for month, cents := range map[string]int64{
		"2025-12": 1000, "2026-01": 2000, "2026-02": 3000,
	} {
		a.addExpense(t, map[string]any{"occurred_on": month + "-10", "amount_cents": cents, "category_id": alimentari.ID})
	}
	before := a.budget(t)

	// A large stock buy in one of the same history months.
	a.addExpense(t, map[string]any{"occurred_on": "2026-01-15", "amount_cents": 500000, "category_id": investments.ID})

	after := a.budget(t)
	if after.BudgetCents != before.BudgetCents {
		t.Errorf("budget_cents = %d after an Investments buy, want unchanged %d", after.BudgetCents, before.BudgetCents)
	}
}

// Target's defining behaviour: it defaults to Budget's own number the first
// time anything reads it, and then holds that number even once Budget itself
// moves — it is the household's own ceiling from then on, not a mirror.
func TestTargetDefaultsToBudgetThenStaysStickyOnceRead(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")

	for month, cents := range map[string]int64{
		"2025-12": 1000, "2026-01": 2000, "2026-02": 3000,
	} {
		a.addExpense(t, map[string]any{"occurred_on": month + "-10", "amount_cents": cents, "category_id": alimentari.ID})
	}

	first := a.budget(t)
	if !first.Available {
		t.Fatalf("available = false, want true: %+v", first)
	}
	if first.TargetCents != first.BudgetCents {
		t.Fatalf("target_cents = %d on first read, want it defaulted to budget_cents %d", first.TargetCents, first.BudgetCents)
	}

	// Move the Budget by adding another month's spend, then read again.
	a.addExpense(t, map[string]any{"occurred_on": "2025-11-10", "amount_cents": 900000, "category_id": alimentari.ID})
	second := a.budget(t)
	if second.BudgetCents == first.BudgetCents {
		t.Fatalf("budget_cents did not move after adding a month's history — test setup is broken")
	}
	if second.TargetCents != first.TargetCents {
		t.Errorf("target_cents = %d after Budget moved, want it to stay at the first-read default %d", second.TargetCents, first.TargetCents)
	}
}

// Target is also explicitly settable through /api/settings, and an explicit
// write is exactly as sticky as the first-read default.
func TestTargetIsSetThroughSettingsAndStaysSet(t *testing.T) {
	a := newTestApp(t)

	if res := a.put(t, settingsPath, map[string]any{"target_cents": int64(75000)}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("PUT %s = %d, want 200", settingsPath, res.StatusCode)
	}

	got := a.budget(t)
	if got.TargetCents != 75000 {
		t.Errorf("target_cents = %d, want the explicitly set 75000", got.TargetCents)
	}
}

// Goal is stored and read back independently of Budget/Target — setting it
// does not touch either.
func TestGoalRoundTripsIndependentlyOfBudgetAndTarget(t *testing.T) {
	a := newTestApp(t)

	before := a.budget(t)

	if res := a.put(t, settingsPath, map[string]any{"goal_cents": int64(50000)}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("PUT %s = %d, want 200", settingsPath, res.StatusCode)
	}

	got := a.budget(t)
	if got.GoalCents != 50000 {
		t.Errorf("goal_cents = %d, want 50000", got.GoalCents)
	}
	if got.BudgetCents != before.BudgetCents || got.TargetCents != before.TargetCents {
		t.Errorf("setting goal_cents moved budget/target: got %+v, before %+v", got, before)
	}
}
