package main

import (
	"net/http"
	"testing"
)

// An Expense as the API hands it out at this ticket's scope: an amount, a
// date, and a Category. The fields that make one recognisable months later —
// Store, Payer, Payment method, note — arrive in 06.
type expenseJSON struct {
	ID          int64  `json:"id"`
	OccurredOn  string `json:"occurred_on"`
	AmountCents int64  `json:"amount_cents"`
	CategoryID  int64  `json:"category_id"`
}

func (a *testApp) expenses(t *testing.T) []expenseJSON {
	t.Helper()
	var got []expenseJSON
	if res := a.get(t, "/api/expenses", &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/expenses = %d, want 200", res.StatusCode)
	}
	return got
}

// addExpense logs one and returns it as the API answered, failing the test if
// the save was refused. Later tickets' tests need a stocked month, and this is
// the one place that knows how to stock it.
func (a *testApp) addExpense(t *testing.T, body map[string]any) expenseJSON {
	t.Helper()
	var created expenseJSON
	res := a.post(t, "/api/expenses", body, &created)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/expenses %v = %d, want 201", body, res.StatusCode)
	}
	return created
}

// The whole ticket in one test: what one request saved is what a later request
// sees, down to the cent.
func TestALoggedExpenseIsThereForTheNextRequest(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")

	created := a.addExpense(t, map[string]any{
		"occurred_on":  "2026-03-15",
		"amount_cents": 4237,
		"category_id":  alimentari.ID,
	})
	if created.ID == 0 {
		t.Error("the created Expense came back without an id")
	}

	got := a.expenses(t)
	if len(got) != 1 {
		t.Fatalf("listed %d Expenses, want 1", len(got))
	}
	if got[0] != created {
		t.Errorf("listed Expense = %+v, want the one that was created, %+v", got[0], created)
	}
	if got[0].AmountCents != 4237 || got[0].OccurredOn != "2026-03-15" || got[0].CategoryID != alimentari.ID {
		t.Errorf("listed Expense = %+v, want 4237 cents on 2026-03-15 in %d", got[0], alimentari.ID)
	}
}

// The list is what the household sees after logging something, so what was
// just spent has to be at the top. Same-day entries fall back to newest first.
func TestExpensesAreListedMostRecentFirst(t *testing.T) {
	a := newTestApp(t)
	id := a.category(t, "Alimentari").ID

	for _, on := range []string{"2026-02-01", "2026-03-15", "2026-01-20", "2026-03-15"} {
		a.addExpense(t, map[string]any{"occurred_on": on, "amount_cents": 100, "category_id": id})
	}

	got := a.expenses(t)
	var dates []string
	for _, e := range got {
		dates = append(dates, e.OccurredOn)
	}
	want := []string{"2026-03-15", "2026-03-15", "2026-02-01", "2026-01-20"}
	if len(dates) != len(want) {
		t.Fatalf("dates = %v, want %v", dates, want)
	}
	for i := range want {
		if dates[i] != want[i] {
			t.Fatalf("dates = %v, want %v", dates, want)
		}
	}
	// The two on the same day: the one logged second is the one listed first.
	if got[0].ID < got[1].ID {
		t.Errorf("same-day ids = %d then %d, want the newer one first", got[0].ID, got[1].ID)
	}
}

// Money is integer cents at the seam and everywhere behind it. A euro amount
// arriving as a float is the bug this closes off at the door: 12.34 cannot be
// represented exactly, and one rounding is all it takes for a month's total to
// stop matching the entries.
func TestAmountsAreIntegerCentsAndFloatsAreRefused(t *testing.T) {
	a := newTestApp(t)
	id := a.category(t, "Alimentari").ID

	created := a.addExpense(t, map[string]any{"occurred_on": "2026-03-15", "amount_cents": 799, "category_id": id})
	if created.AmountCents != 799 {
		t.Errorf("amount_cents = %d, want 799", created.AmountCents)
	}

	res := a.post(t, "/api/expenses", map[string]any{
		"occurred_on":  "2026-03-15",
		"amount_cents": 7.99,
		"category_id":  id,
	}, nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("posting a fractional amount_cents = %d, want 400", res.StatusCode)
	}
	if got := a.expenses(t); len(got) != 1 {
		t.Errorf("listed %d Expenses, want only the valid one", len(got))
	}
}

func TestExpenseWritesAreValidated(t *testing.T) {
	a := newTestApp(t)
	groceries := a.category(t, "Alimentari").ID
	freelance := a.category(t, seedFreelanceName).ID

	cases := map[string]struct {
		body any
		want int
	}{
		"a zero amount":       {map[string]any{"occurred_on": "2026-03-15", "amount_cents": 0, "category_id": groceries}, http.StatusBadRequest},
		"a negative amount":   {map[string]any{"occurred_on": "2026-03-15", "amount_cents": -500, "category_id": groceries}, http.StatusBadRequest},
		"a missing date":      {map[string]any{"amount_cents": 500, "category_id": groceries}, http.StatusBadRequest},
		"a malformed date":    {map[string]any{"occurred_on": "15/03/2026", "amount_cents": 500, "category_id": groceries}, http.StatusBadRequest},
		"an impossible date":  {map[string]any{"occurred_on": "2026-02-30", "amount_cents": 500, "category_id": groceries}, http.StatusBadRequest},
		"no Category":         {map[string]any{"occurred_on": "2026-03-15", "amount_cents": 500}, http.StatusBadRequest},
		"an unknown Category": {map[string]any{"occurred_on": "2026-03-15", "amount_cents": 500, "category_id": 9999}, http.StatusBadRequest},
		"an income Category":  {map[string]any{"occurred_on": "2026-03-15", "amount_cents": 500, "category_id": freelance}, http.StatusBadRequest},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if res := a.post(t, "/api/expenses", c.body, nil); res.StatusCode != c.want {
				t.Errorf("posting %s = %d, want %d", name, res.StatusCode, c.want)
			}
		})
	}

	if got := a.expenses(t); len(got) != 0 {
		t.Errorf("%d refused Expenses were saved anyway: %+v", len(got), got)
	}
}

// A hidden Category is out of the picker, not out of the app: an Expense
// logged against one still saves, which is what makes hiding safe.
func TestAnExpenseCanBeLoggedInAHiddenCategory(t *testing.T) {
	a := newTestApp(t)
	c := a.category(t, "Alimentari")
	a.patch(t, categoryPath(c.ID), map[string]any{"hidden": true}, nil)

	a.addExpense(t, map[string]any{"occurred_on": "2026-03-15", "amount_cents": 500, "category_id": c.ID})
	if got := a.expenses(t); len(got) != 1 {
		t.Errorf("listed %d Expenses, want 1", len(got))
	}
}

// Categories are referenced, not copied, so renaming one fixes every past
// entry at once — the half of ADR-0008's reasoning that needed Expenses to
// exist before it could be observed.
func TestRenamingACategoryLeavesItsExpensesPointingAtIt(t *testing.T) {
	a := newTestApp(t)
	before := a.category(t, "Alimentari")
	logged := a.addExpense(t, map[string]any{"occurred_on": "2026-03-15", "amount_cents": 500, "category_id": before.ID})

	a.patch(t, categoryPath(before.ID), map[string]any{"name": "Cibo"}, nil)

	if after := a.category(t, "Cibo"); after.ID != logged.CategoryID {
		t.Errorf("the renamed Category is id %d, but the Expense points at %d", after.ID, logged.CategoryID)
	}
}
