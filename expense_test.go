package main

import (
	"net/http"
	"strconv"
	"testing"
)

// An Expense as the API hands it out: the three fields that make it an
// Expense, and the four that make it recognisable months later.
type expenseJSON struct {
	ID            int64  `json:"id"`
	OccurredOn    string `json:"occurred_on"`
	AmountCents   int64  `json:"amount_cents"`
	CategoryID    int64  `json:"category_id"`
	Store         string `json:"store"`
	Payer         string `json:"payer"`
	PaymentMethod string `json:"payment_method"`
	Note          string `json:"note"`
}

// expensePath addresses one Expense the way the API does.
func expensePath(id int64) string {
	return "/api/expenses/" + strconv.FormatInt(id, 10)
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

// The four detail fields go in and come back out unchanged. They are what
// makes a €43 line recognisable in November, so a round trip that drops one
// silently is the failure worth pinning.
func TestAnExpenseKeepsItsDetails(t *testing.T) {
	a := newTestApp(t)
	id := a.category(t, "Alimentari").ID

	created := a.addExpense(t, map[string]any{
		"occurred_on":    "2026-03-15",
		"amount_cents":   4237,
		"category_id":    id,
		"store":          "Conad Città",
		"payer":          "Nicco",
		"payment_method": "Bancomat",
		"note":           "spesa grossa, c'era la festa",
	})

	got := a.expenses(t)
	if len(got) != 1 {
		t.Fatalf("listed %d Expenses, want 1", len(got))
	}
	if got[0] != created {
		t.Errorf("listed Expense = %+v, want the created one, %+v", got[0], created)
	}
	if got[0].Store != "Conad Città" || got[0].Payer != "Nicco" ||
		got[0].PaymentMethod != "Bancomat" || got[0].Note != "spesa grossa, c'era la festa" {
		t.Errorf("details came back as %+v", got[0])
	}
}

// Every detail is optional: the three-tap Expense of ticket 05 still saves,
// and reads back as empty text rather than null — the frontend puts these
// straight into inputs.
func TestTheDetailsAreOptional(t *testing.T) {
	a := newTestApp(t)
	id := a.category(t, "Alimentari").ID

	created := a.addExpense(t, map[string]any{"occurred_on": "2026-03-15", "amount_cents": 700, "category_id": id})
	if created.Store != "" || created.Payer != "" || created.PaymentMethod != "" || created.Note != "" {
		t.Errorf("an Expense logged without details = %+v, want the four empty", created)
	}
}

// ADR-0001 makes a Payer a label rather than an identity, and this is what
// that buys: the label is what the entry says, so editing the list it was
// chosen from must not rewrite history. Renaming a Category does the opposite,
// deliberately — see TestRenamingACategoryLeavesItsExpensesPointingAtIt.
func TestRenamingAPayerLeavesExistingExpensesReadingAsBefore(t *testing.T) {
	a := newTestApp(t)
	id := a.category(t, "Alimentari").ID
	before := a.lists(t)

	logged := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 500, "category_id": id,
		"payer": before.Payers[0], "payment_method": before.PaymentMethods[0],
	})

	a.put(t, settingsPath, map[string]any{
		"payers":          []string{"Qualcun altro entirely", before.Payers[1]},
		"payment_methods": []string{"Contanti rinominati"},
	}, nil)

	got := a.expenses(t)
	if len(got) != 1 {
		t.Fatalf("listed %d Expenses, want 1", len(got))
	}
	if got[0].Payer != logged.Payer {
		t.Errorf("payer now reads %q, want the %q it was saved with", got[0].Payer, logged.Payer)
	}
	if got[0].PaymentMethod != logged.PaymentMethod {
		t.Errorf("payment_method now reads %q, want the %q it was saved with", got[0].PaymentMethod, logged.PaymentMethod)
	}
}

// A mistake is a correction, not a delete-and-retype. A PATCH carries only the
// fields that changed; everything else reads back as it was.
func TestAnExpenseCanBeEdited(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID
	svago := a.category(t, "Svago").ID

	logged := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 4237, "category_id": alimentari,
		"store": "Conad", "payer": "Nicco", "payment_method": "Contanti", "note": "sbagliata",
	})

	var updated expenseJSON
	res := a.patch(t, expensePath(logged.ID), map[string]any{
		"amount_cents": 4137,
		"category_id":  svago,
		"note":         "",
	}, &updated)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH %s = %d, want 200", expensePath(logged.ID), res.StatusCode)
	}

	got := a.expenses(t)
	if len(got) != 1 {
		t.Fatalf("listed %d Expenses, want the one, edited", len(got))
	}
	if got[0] != updated {
		t.Errorf("listed Expense = %+v, want what PATCH answered, %+v", got[0], updated)
	}
	want := expenseJSON{
		ID: logged.ID, OccurredOn: "2026-03-15", AmountCents: 4137, CategoryID: svago,
		Store: "Conad", Payer: "Nicco", PaymentMethod: "Contanti",
	}
	if got[0] != want {
		t.Errorf("edited Expense = %+v, want %+v", got[0], want)
	}
}

// An edit is validated exactly as a create is: the same rules, or a bad one
// could be smuggled in through the back door.
func TestEditsAreValidated(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID
	freelance := a.category(t, seedFreelanceName).ID
	logged := a.addExpense(t, map[string]any{"occurred_on": "2026-03-15", "amount_cents": 500, "category_id": alimentari})

	cases := map[string]struct {
		path string
		body any
		want int
	}{
		"a zero amount":       {expensePath(logged.ID), map[string]any{"amount_cents": 0}, http.StatusBadRequest},
		"a fractional amount": {expensePath(logged.ID), map[string]any{"amount_cents": 7.99}, http.StatusBadRequest},
		"a malformed date":    {expensePath(logged.ID), map[string]any{"occurred_on": "15/03/2026"}, http.StatusBadRequest},
		"an income Category":  {expensePath(logged.ID), map[string]any{"category_id": freelance}, http.StatusBadRequest},
		"an unknown Category": {expensePath(logged.ID), map[string]any{"category_id": 9999}, http.StatusBadRequest},
		"an unknown Expense":  {expensePath(9999), map[string]any{"amount_cents": 100}, http.StatusNotFound},
		"an unparseable id":   {"/api/expenses/nope", map[string]any{"amount_cents": 100}, http.StatusNotFound},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if res := a.patch(t, c.path, c.body, nil); res.StatusCode != c.want {
				t.Errorf("patching %s = %d, want %d", name, res.StatusCode, c.want)
			}
		})
	}

	if got := a.expenses(t); len(got) != 1 || got[0] != logged {
		t.Errorf("the Expense reads %+v, want the refused edits to have changed nothing (%+v)", got, logged)
	}
}

// A duplicate entry must not distort the month, so it can go.
func TestAnExpenseCanBeDeleted(t *testing.T) {
	a := newTestApp(t)
	id := a.category(t, "Alimentari").ID
	logged := a.addExpense(t, map[string]any{"occurred_on": "2026-03-15", "amount_cents": 500, "category_id": id})
	kept := a.addExpense(t, map[string]any{"occurred_on": "2026-03-16", "amount_cents": 900, "category_id": id})

	if res := a.delete(t, expensePath(logged.ID)); res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE %s = %d, want 204", expensePath(logged.ID), res.StatusCode)
	}

	got := a.expenses(t)
	if len(got) != 1 || got[0] != kept {
		t.Errorf("after the delete the list is %+v, want only %+v", got, kept)
	}
	if res := a.delete(t, expensePath(logged.ID)); res.StatusCode != http.StatusNotFound {
		t.Errorf("deleting it twice = %d, want 404", res.StatusCode)
	}
}
