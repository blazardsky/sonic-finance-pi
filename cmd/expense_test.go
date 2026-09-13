package main

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"testing"
)

// An Expense as the API hands it out: the three fields that make it an
// Expense, the four that make it recognisable months later, and the optional
// partial breakdown of ticket 07.
type expenseJSON struct {
	ID            int64      `json:"id"`
	OccurredOn    string     `json:"occurred_on"`
	AmountCents   int64      `json:"amount_cents"`
	CategoryID    int64      `json:"category_id"`
	Store         string     `json:"store"`
	Payer         string     `json:"payer"`
	PaymentMethod string     `json:"payment_method"`
	Note          string     `json:"note"`
	Items         []itemJSON `json:"items"`

	// The year a tax payment relates to, and 0 on every Expense that is not
	// one. Ticket 15: tax on 2026's income is paid during 2027, so the year
	// it is attributed to is not the year it left the account.
	TaxYear int `json:"tax_year"`

	// Ticket 03: the Holding a buy Expense is for, nil on every other Expense.
	HoldingID *int64 `json:"holding_id"`

	// A second, independent tag alongside CategoryID, nil on most Expenses.
	SubcategoryID *int64 `json:"subcategory_id"`
}

// An Item as the API hands it out. It carries no id: nothing addresses an Item
// on its own, so what a request sends is exactly what a later one reads back.
//
// Ticket 01: Quantity/Unit are the optional pair, PricePerUnit is the derived,
// read-only figure computed from them (ADR-0014), and Discounted is the
// purely informational flag.
type itemJSON struct {
	Name        string   `json:"name"`
	AmountCents int64    `json:"amount_cents"`
	CategoryID  int64    `json:"category_id"`
	Quantity    *float64 `json:"quantity"`
	Unit        string   `json:"unit"`
	Discounted  bool     `json:"discounted"`

	PricePerUnit *float64 `json:"price_per_unit"`
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
//
// Payer defaults here when a case does not care about it, now that ticket 08
// makes it required: the alternative is threading a payer through every one
// of this file's callers, most of which are testing something else entirely.
func (a *testApp) addExpense(t *testing.T, body map[string]any) expenseJSON {
	t.Helper()
	if _, ok := body["payer"]; !ok {
		body["payer"] = "Nicco"
	}
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
	if !reflect.DeepEqual(got[0], created) {
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
		"an unknown subcategory": {map[string]any{"occurred_on": "2026-03-15", "amount_cents": 500, "category_id": groceries, "subcategory_id": 9999}, http.StatusBadRequest},
		// Ticket 08: "whose money was it" is never left unanswered going forward.
		"an empty payer":     {map[string]any{"occurred_on": "2026-03-15", "amount_cents": 500, "category_id": groceries, "payer": ""}, http.StatusBadRequest},
		"a whitespace payer": {map[string]any{"occurred_on": "2026-03-15", "amount_cents": 500, "category_id": groceries, "payer": "   "}, http.StatusBadRequest},
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
	if !reflect.DeepEqual(got[0], created) {
		t.Errorf("listed Expense = %+v, want the created one, %+v", got[0], created)
	}
	if got[0].Store != "Conad Città" || got[0].Payer != "Nicco" ||
		got[0].PaymentMethod != "Bancomat" || got[0].Note != "spesa grossa, c'era la festa" {
		t.Errorf("details came back as %+v", got[0])
	}
}

// Three of the four details are optional: the three-tap Expense of ticket 05
// still saves, and reads back as empty text rather than null — the frontend
// puts these straight into inputs. Payer is the deliberate exception since
// ticket 08 — see TestExpenseWritesAreValidated and TestEditsAreValidated.
func TestTheDetailsAreOptional(t *testing.T) {
	a := newTestApp(t)
	id := a.category(t, "Alimentari").ID

	created := a.addExpense(t, map[string]any{"occurred_on": "2026-03-15", "amount_cents": 700, "category_id": id, "payer": "Nicco"})
	if created.Store != "" || created.PaymentMethod != "" || created.Note != "" {
		t.Errorf("an Expense logged without details = %+v, want the three empty", created)
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
	if !reflect.DeepEqual(got[0], updated) {
		t.Errorf("listed Expense = %+v, want what PATCH answered, %+v", got[0], updated)
	}
	want := expenseJSON{
		ID: logged.ID, OccurredOn: "2026-03-15", AmountCents: 4137, CategoryID: svago,
		Store: "Conad", Payer: "Nicco", PaymentMethod: "Contanti",
		Items: []itemJSON{},
	}
	if !reflect.DeepEqual(got[0], want) {
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
		"an unknown subcategory": {expensePath(logged.ID), map[string]any{"subcategory_id": 9999}, http.StatusBadRequest},
		"an unknown Expense":  {expensePath(9999), map[string]any{"amount_cents": 100}, http.StatusNotFound},
		"an unparseable id":   {"/api/expenses/nope", map[string]any{"amount_cents": 100}, http.StatusNotFound},
		// Ticket 08: an edit setting the Payer empty is refused the same as a create.
		"an empty payer":     {expensePath(logged.ID), map[string]any{"payer": ""}, http.StatusBadRequest},
		"a whitespace payer": {expensePath(logged.ID), map[string]any{"payer": "   "}, http.StatusBadRequest},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if res := a.patch(t, c.path, c.body, nil); res.StatusCode != c.want {
				t.Errorf("patching %s = %d, want %d", name, res.StatusCode, c.want)
			}
		})
	}

	if got := a.expenses(t); len(got) != 1 || !reflect.DeepEqual(got[0], logged) {
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
	if len(got) != 1 || !reflect.DeepEqual(got[0], kept) {
		t.Errorf("after the delete the list is %+v, want only %+v", got, kept)
	}
	if res := a.delete(t, expensePath(logged.ID)); res.StatusCode != http.StatusNotFound {
		t.Errorf("deleting it twice = %d, want 404", res.StatusCode)
	}
}

// The ticket's own example: a book bought during the grocery shop stops
// counting as Food, without the receipt ever being fully itemised. The €62
// shop stays a €62 shop — ADR-0002 — and €14 of it now belongs to Svago.
func TestPartOfAnExpenseCanBeBrokenOutUnderAnotherCategory(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID
	svago := a.category(t, "Svago").ID

	created := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 6200, "category_id": alimentari,
		"items": []map[string]any{{"name": "Libro", "amount_cents": 1400, "category_id": svago}},
	})

	if created.AmountCents != 6200 {
		t.Errorf("amount_cents = %d, want the authoritative 6200 the receipt said", created.AmountCents)
	}
	want := []itemJSON{{Name: "Libro", AmountCents: 1400, CategoryID: svago}}
	if !reflect.DeepEqual(created.Items, want) {
		t.Errorf("items = %+v, want %+v", created.Items, want)
	}

	got := a.expenses(t)
	if len(got) != 1 || !reflect.DeepEqual(got[0], created) {
		t.Errorf("listed Expense = %+v, want the created one, %+v", got, created)
	}
}

// Items never have to account for the whole Expense, and several can sit under
// one: what they do not cover stays under the Expense's own Category, which is
// the remainder ticket 11 will report on.
func TestItemsAreAPartialBreakdownInTheOrderTheyWereSent(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID
	svago := a.category(t, "Svago").ID
	salute := a.category(t, "Salute").ID

	created := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 6200, "category_id": alimentari,
		"items": []map[string]any{
			{"name": "Libro", "amount_cents": 1400, "category_id": svago},
			{"name": "Aspirina", "amount_cents": 600, "category_id": salute},
		},
	})

	want := []itemJSON{
		{Name: "Libro", AmountCents: 1400, CategoryID: svago},
		{Name: "Aspirina", AmountCents: 600, CategoryID: salute},
	}
	if !reflect.DeepEqual(created.Items, want) {
		t.Errorf("items = %+v, want %+v", created.Items, want)
	}
}

// A €7 coffee is one record and nothing else. Items read back as an empty list
// rather than null, because the frontend maps over them.
func TestAnExpenseWithoutItemsStaysASingleRecord(t *testing.T) {
	a := newTestApp(t)
	id := a.category(t, "Alimentari").ID

	created := a.addExpense(t, map[string]any{"occurred_on": "2026-03-15", "amount_cents": 700, "category_id": id})
	if created.Items == nil || len(created.Items) != 0 {
		t.Errorf("items = %#v, want an empty list", created.Items)
	}
	if got := a.expenses(t); len(got) != 1 || len(got[0].Items) != 0 {
		t.Errorf("listed Expense = %+v, want it with no items", got)
	}
}

// A negative remainder has no meaning, so the save is refused — and refused
// with a sentence the frontend can put in front of whoever is standing in the
// shop, not a bare "Bad Request".
func TestItemsAddingUpToMoreThanTheExpenseAreRejected(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID
	svago := a.category(t, "Svago").ID

	var body struct{ Error string }
	res := a.post(t, "/api/expenses", map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 2000, "category_id": alimentari,
		"items": []map[string]any{
			{"name": "Libro", "amount_cents": 1400, "category_id": svago},
			{"name": "Rivista", "amount_cents": 900, "category_id": svago},
		},
	}, &body)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("posting items over the total = %d, want 400", res.StatusCode)
	}
	if body.Error == "" || body.Error == http.StatusText(http.StatusBadRequest) {
		t.Errorf("error = %q, want a message saying what was wrong", body.Error)
	}

	if got := a.expenses(t); len(got) != 0 {
		t.Errorf("the refused Expense was saved anyway: %+v", got)
	}
}

// Items exactly covering the Expense is the boundary, and it is allowed: a
// fully itemised receipt leaves a remainder of zero, not a negative one.
func TestItemsMayCoverTheWholeExpense(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID
	svago := a.category(t, "Svago").ID

	created := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 2000, "category_id": alimentari,
		"items": []map[string]any{
			{"name": "Libro", "amount_cents": 1400, "category_id": svago},
			{"name": "Rivista", "amount_cents": 600, "category_id": svago},
		},
	})
	if len(created.Items) != 2 {
		t.Errorf("items = %+v, want both saved", created.Items)
	}
}

// Pointless but permitted, and deliberately unvalidated: deciding for someone
// that they may not itemise within their own Category would be a rule with
// nothing behind it.
func TestAnItemMayShareTheExpensesCategory(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID

	created := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 6200, "category_id": alimentari,
		"items": []map[string]any{{"name": "Pane", "amount_cents": 200, "category_id": alimentari}},
	})
	if len(created.Items) != 1 {
		t.Errorf("items = %+v, want the one sharing the Expense's Category", created.Items)
	}
}

// An Item is an Expense's worth of money in a Category, so it answers to the
// same rules the Expense does: a real amount, and a Category an Expense can go
// in. It needs a name too, since telling two Items apart is the whole point.
func TestItemsAreValidated(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID
	svago := a.category(t, "Svago").ID
	freelance := a.category(t, seedFreelanceName).ID

	item := func(over map[string]any) map[string]any {
		it := map[string]any{"name": "Libro", "amount_cents": 1400, "category_id": svago}
		for k, v := range over {
			it[k] = v
		}
		return map[string]any{
			"occurred_on": "2026-03-15", "amount_cents": 6200, "category_id": alimentari,
			"items": []map[string]any{it},
		}
	}

	cases := map[string]any{
		"a missing name":          item(map[string]any{"name": nil}),
		"a blank name":            item(map[string]any{"name": "   "}),
		"a zero amount":           item(map[string]any{"amount_cents": 0}),
		"a negative amount":       item(map[string]any{"amount_cents": -100}),
		"a fractional amount":     item(map[string]any{"amount_cents": 14.5}),
		"no Category":             item(map[string]any{"category_id": nil}),
		"an unknown Category":     item(map[string]any{"category_id": 9999}),
		"an income-only Category": item(map[string]any{"category_id": freelance}),
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if res := a.post(t, "/api/expenses", body, nil); res.StatusCode != http.StatusBadRequest {
				t.Errorf("posting an item with %s = %d, want 400", name, res.StatusCode)
			}
		})
	}

	if got := a.expenses(t); len(got) != 0 {
		t.Errorf("%d refused Expenses were saved anyway: %+v", len(got), got)
	}
}

// Items are saved with the Expense in one request, so they are edited with it
// too: a body carrying items replaces the whole breakdown, one omitting them
// leaves it alone, and an empty list is how a breakdown is removed.
func TestItemsAreEditedWithTheirExpense(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID
	svago := a.category(t, "Svago").ID
	salute := a.category(t, "Salute").ID

	logged := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 6200, "category_id": alimentari,
		"items": []map[string]any{{"name": "Libro", "amount_cents": 1400, "category_id": svago}},
	})

	// A breakdown sent in full replaces what was there.
	var updated expenseJSON
	res := a.patch(t, expensePath(logged.ID), map[string]any{
		"items": []map[string]any{{"name": "Aspirina", "amount_cents": 600, "category_id": salute}},
	}, &updated)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH %s = %d, want 200", expensePath(logged.ID), res.StatusCode)
	}
	want := []itemJSON{{Name: "Aspirina", AmountCents: 600, CategoryID: salute}}
	if !reflect.DeepEqual(updated.Items, want) {
		t.Fatalf("items after the edit = %+v, want %+v", updated.Items, want)
	}

	// An edit that says nothing about items leaves them standing.
	a.patch(t, expensePath(logged.ID), map[string]any{"store": "Conad"}, &updated)
	if !reflect.DeepEqual(updated.Items, want) {
		t.Errorf("items after an unrelated edit = %+v, want the untouched %+v", updated.Items, want)
	}

	// An empty list is how the breakdown goes away.
	a.patch(t, expensePath(logged.ID), map[string]any{"items": []map[string]any{}}, &updated)
	if len(updated.Items) != 0 {
		t.Errorf("items after clearing = %+v, want none", updated.Items)
	}
	if got := a.expenses(t); len(got) != 1 || len(got[0].Items) != 0 {
		t.Errorf("listed Expense = %+v, want the cleared breakdown to have stuck", got)
	}
}

// The Expense total stays authoritative under editing too: it cannot be
// corrected downwards past what its own Items already claim, and a refused
// edit leaves both the Expense and its breakdown as they were.
func TestAnExpenseCannotBeEditedBelowItsItems(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID
	svago := a.category(t, "Svago").ID

	logged := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 6200, "category_id": alimentari,
		"items": []map[string]any{{"name": "Libro", "amount_cents": 1400, "category_id": svago}},
	})

	cases := map[string]any{
		"a total under its items": map[string]any{"amount_cents": 1000},
		"items over the total":    map[string]any{"items": []map[string]any{{"name": "TV", "amount_cents": 90000, "category_id": svago}}},
		"an invalid item":         map[string]any{"items": []map[string]any{{"name": "", "amount_cents": 100, "category_id": svago}}},
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if res := a.patch(t, expensePath(logged.ID), body, nil); res.StatusCode != http.StatusBadRequest {
				t.Errorf("patching %s = %d, want 400", name, res.StatusCode)
			}
		})
	}

	if got := a.expenses(t); len(got) != 1 || !reflect.DeepEqual(got[0], logged) {
		t.Errorf("the Expense reads %+v, want the refused edits to have changed nothing (%+v)", got, logged)
	}
}

// An Item has no life outside its Expense, so deleting the Expense has to take
// the breakdown with it rather than fail on it.
func TestDeletingAnExpenseTakesItsItemsWithIt(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID
	svago := a.category(t, "Svago").ID

	logged := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 6200, "category_id": alimentari,
		"items": []map[string]any{{"name": "Libro", "amount_cents": 1400, "category_id": svago}},
	})

	if res := a.delete(t, expensePath(logged.ID)); res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE %s = %d, want 204", expensePath(logged.ID), res.StatusCode)
	}
	if got := a.expenses(t); len(got) != 0 {
		t.Errorf("after the delete the list is %+v, want it empty", got)
	}
}

// Two Expenses' breakdowns must not bleed into each other — the list loads
// every Item in one query and groups them, and grouping is where that goes
// wrong.
func TestEachExpenseKeepsItsOwnItems(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID
	svago := a.category(t, "Svago").ID

	first := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-14", "amount_cents": 6200, "category_id": alimentari,
		"items": []map[string]any{{"name": "Libro", "amount_cents": 1400, "category_id": svago}},
	})
	second := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 700, "category_id": alimentari,
	})
	third := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-16", "amount_cents": 3000, "category_id": alimentari,
		"items": []map[string]any{
			{"name": "Cuffie", "amount_cents": 500, "category_id": svago},
			{"name": "Rivista", "amount_cents": 400, "category_id": svago},
		},
	})

	got := a.expenses(t)
	if len(got) != 3 {
		t.Fatalf("listed %d Expenses, want 3", len(got))
	}
	for _, want := range []expenseJSON{third, second, first} {
		var found *expenseJSON
		for i := range got {
			if got[i].ID == want.ID {
				found = &got[i]
			}
		}
		if found == nil {
			t.Fatalf("Expense %d is missing from %+v", want.ID, got)
		}
		if !reflect.DeepEqual(*found, want) {
			t.Errorf("Expense %d reads %+v, want %+v", want.ID, *found, want)
		}
	}
}

// expense reads one Expense back out of the list, or fails the test. There is
// no GET for a single one — the list is how the screens read them — so this is
// what "what a later request sees" means for one row.
func (a *testApp) expense(t *testing.T, id int64) expenseJSON {
	t.Helper()
	for _, e := range a.expenses(t) {
		if e.ID == id {
			return e
		}
	}
	t.Fatalf("no Expense with id %d in the list", id)
	return expenseJSON{}
}

// taxes is the Expense-side Base category, and the only one a Tax year is
// kept on.
func (a *testApp) taxes(t *testing.T) categoryJSON {
	t.Helper()
	return a.category(t, seedTaxesName)
}

// investments is the promoted Base category a buy/sell is recorded under
// (ticket 03) — applies_to 'both', so it works from both this file and
// income_test.go.
func (a *testApp) investments(t *testing.T) categoryJSON {
	t.Helper()
	return a.category(t, seedInvestmentiName)
}

// The default that makes the common case free: tax paid in a year is tax on
// that year until someone says otherwise, so nothing has to be typed for the
// payment that lands in the year it belongs to.
func TestATaxExpenseDefaultsItsTaxYearToTheYearItWasPaid(t *testing.T) {
	a := newTestApp(t)

	created := a.addExpense(t, map[string]any{
		"occurred_on": "2027-06-30", "amount_cents": 250000,
		"category_id": a.taxes(t).ID,
	})
	if created.TaxYear != 2027 {
		t.Errorf("tax_year = %d, want 2027 — the default is the year of the payment", created.TaxYear)
	}
	if stored := a.expense(t, created.ID); stored.TaxYear != 2027 {
		t.Errorf("stored tax_year = %d, want 2027", stored.TaxYear)
	}
}

// And the real case, which is the reason the field exists at all: the balance
// paid in June 2027 is tax on 2026's income, and has to say so.
func TestATaxYearIsEditableAndKeptAsTyped(t *testing.T) {
	a := newTestApp(t)
	taxes := a.taxes(t)

	created := a.addExpense(t, map[string]any{
		"occurred_on": "2027-06-30", "amount_cents": 250000,
		"category_id": taxes.ID, "tax_year": 2026,
	})
	if created.TaxYear != 2026 {
		t.Fatalf("tax_year on create = %d, want 2026", created.TaxYear)
	}

	if res := a.patch(t, expensePath(created.ID), map[string]any{"tax_year": 2025}, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH tax_year = %d, want 200", res.StatusCode)
	}
	if stored := a.expense(t, created.ID); stored.TaxYear != 2025 {
		t.Errorf("tax_year after edit = %d, want 2025", stored.TaxYear)
	}

	// An edit about something else leaves it alone: a PATCH merges onto the
	// stored row, and correcting an amount is not a statement about the year.
	a.patch(t, expensePath(created.ID), map[string]any{"amount_cents": 260000}, nil)
	if stored := a.expense(t, created.ID); stored.TaxYear != 2025 {
		t.Errorf("tax_year after an unrelated edit = %d, want 2025", stored.TaxYear)
	}
}

// The field is a tax Category's, and nobody else's: the screens surface it
// only there, and the server does not keep a year on a shopping trip that
// sent one anyway.
func TestAnExpenseOutsideATaxCategoryCarriesNoTaxYear(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari")

	created := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-02", "amount_cents": 4237,
		"category_id": alimentari.ID, "tax_year": 2025,
	})
	if created.TaxYear != 0 {
		t.Errorf("tax_year = %d, want 0 — only a tax expense has one", created.TaxYear)
	}
}

// Moving an Expense out of the tax Category takes the year with it, and
// moving one in gives it the default. The alternative is a stale year sitting
// on a row nothing reads it from, waiting to be wrong if it is moved back.
func TestATaxYearFollowsTheCategoryTheExpenseIsMovedTo(t *testing.T) {
	a := newTestApp(t)
	taxes, alimentari := a.taxes(t), a.category(t, "Alimentari")

	created := a.addExpense(t, map[string]any{
		"occurred_on": "2027-06-30", "amount_cents": 250000,
		"category_id": taxes.ID, "tax_year": 2026,
	})

	a.patch(t, expensePath(created.ID), map[string]any{"category_id": alimentari.ID}, nil)
	if stored := a.expense(t, created.ID); stored.TaxYear != 0 {
		t.Errorf("tax_year after moving out of the tax category = %d, want 0", stored.TaxYear)
	}

	a.patch(t, expensePath(created.ID), map[string]any{"category_id": taxes.ID}, nil)
	if stored := a.expense(t, created.ID); stored.TaxYear != 2027 {
		t.Errorf("tax_year after moving back = %d, want 2027 — the default is the payment year", stored.TaxYear)
	}
}

// A year that is not one is refused rather than stored: the column takes any
// integer, and a 26 typed for 2026 would silently attribute the payment to
// nothing at all.
func TestATaxYearThatIsNotAYearIsRefused(t *testing.T) {
	a := newTestApp(t)

	res := a.post(t, "/api/expenses", map[string]any{
		"occurred_on": "2027-06-30", "amount_cents": 250000,
		"category_id": a.taxes(t).ID, "tax_year": 26,
	}, nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("POST with tax_year 26 = %d, want 400", res.StatusCode)
	}
}

// Ticket 03: buying a Holding is an ordinary Expense with a holding_id set —
// the whole ticket's backend half in one round trip.
func TestABuyExpenseKeepsItsHoldingID(t *testing.T) {
	a := newTestApp(t)
	vwce := a.createHolding(t, "VWCE", holdingETF)

	created := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 20000,
		"category_id": a.investments(t).ID, "holding_id": vwce.ID,
	})
	if created.HoldingID == nil || *created.HoldingID != vwce.ID {
		t.Fatalf("holding_id = %v, want %d", created.HoldingID, vwce.ID)
	}

	if stored := a.expense(t, created.ID); stored.HoldingID == nil || *stored.HoldingID != vwce.ID {
		t.Errorf("stored holding_id = %v, want %d", stored.HoldingID, vwce.ID)
	}
}

// An ordinary Expense carries no Holding at all — nil, not a made-up id.
func TestAnOrdinaryExpenseCarriesNoHoldingID(t *testing.T) {
	a := newTestApp(t)

	created := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 4237,
		"category_id": a.category(t, "Alimentari").ID,
	})
	if created.HoldingID != nil {
		t.Errorf("holding_id = %v, want nil", *created.HoldingID)
	}
}

// A PATCH can set, change, or clear the Holding a buy Expense points at, the
// same partial bargain client_id makes on an Income.
func TestAHoldingIDCanBeSetChangedAndClearedByPatch(t *testing.T) {
	a := newTestApp(t)
	investments := a.investments(t).ID
	vwce := a.createHolding(t, "VWCE", holdingETF)
	btc := a.createHolding(t, "BTC", holdingCrypto)

	logged := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 20000, "category_id": investments,
	})
	if logged.HoldingID != nil {
		t.Fatalf("holding_id = %v, want nil before it is set", logged.HoldingID)
	}

	var got expenseJSON
	a.patch(t, expensePath(logged.ID), map[string]any{"holding_id": vwce.ID}, &got)
	if got.HoldingID == nil || *got.HoldingID != vwce.ID {
		t.Fatalf("holding_id after setting it = %v, want %d", got.HoldingID, vwce.ID)
	}

	a.patch(t, expensePath(logged.ID), map[string]any{"holding_id": btc.ID}, &got)
	if got.HoldingID == nil || *got.HoldingID != btc.ID {
		t.Fatalf("holding_id after changing it = %v, want %d", got.HoldingID, btc.ID)
	}

	// An edit about something else leaves it alone: a PATCH merges onto the
	// stored row.
	a.patch(t, expensePath(logged.ID), map[string]any{"amount_cents": 21000}, &got)
	if got.HoldingID == nil || *got.HoldingID != btc.ID {
		t.Errorf("holding_id after an unrelated edit = %v, want it still %d", got.HoldingID, btc.ID)
	}

	a.patch(t, expensePath(logged.ID), map[string]any{"holding_id": nil}, &got)
	if got.HoldingID != nil {
		t.Errorf("holding_id after clearing it = %v, want nil", got.HoldingID)
	}
}

// ptr is a float64 literal address, for building an itemJSON's optional
// Quantity in table-driven cases without a named variable per case.
func ptr(f float64) *float64 { return &f }

// Ticket 01: quantity and unit are a pair. Either alone is meaningless — a
// quantity with no unit does not say what it counts, and a unit with no
// quantity has nothing to divide by — so both present or both absent is the
// only shape accepted.
func TestItemQuantityAndUnitMustBothBePresentOrBothAbsent(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID
	svago := a.category(t, "Svago").ID

	item := func(over map[string]any) map[string]any {
		it := map[string]any{"name": "Mele", "amount_cents": 500, "category_id": svago}
		for k, v := range over {
			it[k] = v
		}
		return map[string]any{
			"occurred_on": "2026-03-15", "amount_cents": 6200, "category_id": alimentari,
			"items": []map[string]any{it},
		}
	}

	cases := map[string]any{
		"a quantity with no unit": item(map[string]any{"quantity": 1.5}),
		"a unit with no quantity": item(map[string]any{"unit": "kg"}),
		"an invalid unit":         item(map[string]any{"quantity": 1.5, "unit": "grams"}),
		"a zero quantity":         item(map[string]any{"quantity": 0, "unit": "kg"}),
		"a negative quantity":     item(map[string]any{"quantity": -1, "unit": "kg"}),
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if res := a.post(t, "/api/expenses", body, nil); res.StatusCode != http.StatusBadRequest {
				t.Errorf("posting an item with %s = %d, want 400", name, res.StatusCode)
			}
		})
	}

	if got := a.expenses(t); len(got) != 0 {
		t.Errorf("%d refused Expenses were saved anyway: %+v", len(got), got)
	}
}

// The whole ticket: a price per unit is derived from what was actually paid,
// never stored, and never trusted from a request — ADR-0014. A request that
// tries to set price_per_unit directly has it silently overwritten with the
// real computation instead of being saved as sent.
func TestItemPricePerUnitIsComputedAndReadOnly(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID
	svago := a.category(t, "Svago").ID

	created := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 6200, "category_id": alimentari,
		"items": []map[string]any{{
			"name": "Mele", "amount_cents": 500, "category_id": svago,
			"quantity": 2.0, "unit": "kg",
			"price_per_unit": 999999, // a lie the server must not repeat back
		}},
	})

	if len(created.Items) != 1 {
		t.Fatalf("items = %+v, want 1", created.Items)
	}
	it := created.Items[0]
	if it.Quantity == nil || *it.Quantity != 2.0 || it.Unit != "kg" {
		t.Fatalf("item = %+v, want quantity 2.0 kg", it)
	}
	if it.PricePerUnit == nil || *it.PricePerUnit != 250 {
		t.Fatalf("price_per_unit = %v, want 250 (500 cents / 2kg)", it.PricePerUnit)
	}

	got := a.expenses(t)
	if len(got) != 1 || len(got[0].Items) != 1 {
		t.Fatalf("listed = %+v, want the one Item back", got)
	}
	if listed := got[0].Items[0]; listed.PricePerUnit == nil || *listed.PricePerUnit != 250 {
		t.Errorf("listed price_per_unit = %v, want 250", listed.PricePerUnit)
	}
}

// An Item without a quantity has nothing to divide by, so it carries no
// price_per_unit at all — nil, not zero.
func TestItemPricePerUnitIsNilWithoutAQuantity(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID
	svago := a.category(t, "Svago").ID

	created := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 6200, "category_id": alimentari,
		"items": []map[string]any{{"name": "Libro", "amount_cents": 1400, "category_id": svago}},
	})
	if created.Items[0].PricePerUnit != nil {
		t.Errorf("price_per_unit = %v, want nil without a quantity", *created.Items[0].PricePerUnit)
	}
}

// Discounted is purely informational (ADR-0014): it defaults to false, and
// round-trips as sent without touching amount, quantity, or the derived price.
func TestItemDiscountedFlagRoundTripsAndDefaultsFalse(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID
	svago := a.category(t, "Svago").ID

	created := a.addExpense(t, map[string]any{
		"occurred_on": "2026-03-15", "amount_cents": 6200, "category_id": alimentari,
		"items": []map[string]any{
			{"name": "Mele", "amount_cents": 500, "category_id": svago, "quantity": 2.0, "unit": "kg", "discounted": true},
			{"name": "Libro", "amount_cents": 1400, "category_id": svago},
		},
	})

	if !created.Items[0].Discounted {
		t.Errorf("discounted = %v, want true — it was sent that way", created.Items[0].Discounted)
	}
	if created.Items[1].Discounted {
		t.Errorf("discounted = %v, want false by default", created.Items[1].Discounted)
	}
	if created.Items[0].PricePerUnit == nil || *created.Items[0].PricePerUnit != 250 {
		t.Errorf("discounted must not change the derived price: got %v, want 250", created.Items[0].PricePerUnit)
	}
}

// A bare GET must keep returning every Expense unfiltered — Investments.tsx
// reads its whole buy/sell history off this same endpoint with no query
// params at all, and a default that quietly capped or filtered it would
// break that page without ever touching its own code.
func TestListingExpensesWithNoParamsIsUnfiltered(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID
	a.addExpense(t, map[string]any{"occurred_on": "2020-01-01", "amount_cents": 100, "category_id": alimentari})
	a.addExpense(t, map[string]any{"occurred_on": "2026-01-01", "amount_cents": 200, "category_id": alimentari})

	got := a.expenses(t)
	if len(got) != 2 {
		t.Fatalf("len(expenses) = %d, want 2 — no params must mean no filtering", len(got))
	}
}

func TestYearFiltersExpensesToThatYearOnly(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID
	a.addExpense(t, map[string]any{"occurred_on": "2025-06-01", "amount_cents": 100, "category_id": alimentari})
	a.addExpense(t, map[string]any{"occurred_on": "2026-01-01", "amount_cents": 200, "category_id": alimentari})
	a.addExpense(t, map[string]any{"occurred_on": "2026-12-31", "amount_cents": 300, "category_id": alimentari})

	var got []expenseJSON
	if res := a.get(t, "/api/expenses?year=2026", &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET ?year=2026 = %d, want 200", res.StatusCode)
	}
	if len(got) != 2 {
		t.Fatalf("len(expenses) = %d, want 2 (only 2026)", len(got))
	}
	for _, e := range got {
		if e.OccurredOn[:4] != "2026" {
			t.Errorf("occurred_on = %q, want a 2026 date", e.OccurredOn)
		}
	}
}

func TestLimitCapsHowManyExpensesComeBack(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID
	for i := 1; i <= 5; i++ {
		a.addExpense(t, map[string]any{
			"occurred_on": fmt.Sprintf("2026-01-0%d", i), "amount_cents": 100, "category_id": alimentari,
		})
	}

	var got []expenseJSON
	if res := a.get(t, "/api/expenses?limit=3", &got); res.StatusCode != http.StatusOK {
		t.Fatalf("GET ?limit=3 = %d, want 200", res.StatusCode)
	}
	if len(got) != 3 {
		t.Fatalf("len(expenses) = %d, want 3", len(got))
	}
	// Newest first, same ordering the unfiltered list already uses.
	if got[0].OccurredOn != "2026-01-05" {
		t.Errorf("first row = %q, want the newest (2026-01-05)", got[0].OccurredOn)
	}
}

func TestOffsetPagesPastAlreadySeenExpenses(t *testing.T) {
	a := newTestApp(t)
	alimentari := a.category(t, "Alimentari").ID
	for i := 1; i <= 5; i++ {
		a.addExpense(t, map[string]any{
			"occurred_on": fmt.Sprintf("2026-01-0%d", i), "amount_cents": 100, "category_id": alimentari,
		})
	}

	var page1, page2 []expenseJSON
	a.get(t, "/api/expenses?limit=2&offset=0", &page1)
	a.get(t, "/api/expenses?limit=2&offset=2", &page2)

	if len(page1) != 2 || len(page2) != 2 {
		t.Fatalf("page1 = %d, page2 = %d, want 2 and 2", len(page1), len(page2))
	}
	for _, p1 := range page1 {
		for _, p2 := range page2 {
			if p1.ID == p2.ID {
				t.Errorf("expense %d appears on both pages", p1.ID)
			}
		}
	}
}

func TestAMalformedYearIsRefused(t *testing.T) {
	a := newTestApp(t)
	for _, year := range []string{"26", "20266", "abcd"} {
		res := a.get(t, "/api/expenses?year="+year, nil)
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("year=%q: status = %d, want 400", year, res.StatusCode)
		}
	}
}

func TestAMalformedLimitOrOffsetIsRefused(t *testing.T) {
	a := newTestApp(t)
	for _, qs := range []string{"limit=0", "limit=-1", "limit=abc", "limit=5&offset=-1", "limit=5&offset=abc"} {
		res := a.get(t, "/api/expenses?"+qs, nil)
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", qs, res.StatusCode)
		}
	}
}
